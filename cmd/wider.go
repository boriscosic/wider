package main

import (
	"context"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"strings"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

type PodWithWider struct {
	Pod            *corev1.Pod
	Node           *corev1.Node
	ServiceAccount *corev1.ServiceAccount
	PVCs           []*corev1.PersistentVolumeClaim
}

type Options struct {
	Namespace     string
	OutputFormat  string
	AllNamespaces bool
	Clientset     *kubernetes.Clientset
	ConfigFlags   *clientcmd.ClientConfigLoadingRules
}

func (o *Options) Complete() error {
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(o.ConfigFlags, configOverrides)

	config, err := kubeConfig.ClientConfig()
	if err != nil {
		return fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	o.Clientset, err = kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	// Get current namespace if not specified
	if o.Namespace == "" && !o.AllNamespaces {
		o.Namespace, _, err = kubeConfig.Namespace()
		if err != nil {
			return fmt.Errorf("failed to get current namespace: %w", err)
		}
	}

	return nil
}

func NewWiderOptions() *Options {
	return &Options{
		ConfigFlags: clientcmd.NewDefaultClientConfigLoadingRules(),
	}
}

func NewRootCommand() *cobra.Command {
	opts := NewWiderOptions()

	cmd := &cobra.Command{
		Use:   "kubectl-wider",
		Short: "Get pods with extended node information",
		Long: `kubectl-wider retrieves pods and extends them with corresponding information, supports owner/controller, node, service account and pvc.
		
Examples:
  # List pods with node info in current namespace
  kubectl wider
  
  # List pods in specific namespace
  kubectl wider -n kube-system
  
  # List pods in all namespaces
  kubectl wider -A

  # Support for nodes, serviceAccount, pvc
  kubectl wider -A

  # Custom columns output
  kubectl wider -o custom-columns=NAME:.pod.metadata.name,NODE:.node.metadata.name,OS:.node.metadata.labels.kubernetes\.io/os

  More information is available at the project website:
  https://github.com/boriscosic/wider`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Complete(); err != nil {
				return err
			}
			if err := opts.Validate(); err != nil {
				return err
			}
			return opts.Run()
		},
	}

	cmd.Flags().StringVarP(&opts.Namespace, "namespace", "n", "", "Namespace to query (defaults to current context namespace)")
	cmd.Flags().StringVarP(&opts.OutputFormat, "output", "o", "", "Output format. One of: (json, yaml, custom-columns) (e.g., custom-columns=\"NAME:.pod.metadata.name,NODE:.node.metadata.name,OS:.node.metadata.labels.kubernetes\\.io/os\")")
	cmd.Flags().BoolVarP(&opts.AllNamespaces, "all-namespaces", "A", false, "Query all namespaces")

	return cmd
}

func (o *Options) Validate() error {
	if o.OutputFormat != "" && !strings.HasPrefix(o.OutputFormat, "custom-columns=") {
		return fmt.Errorf("unsupported output format: %s (only custom-columns is supported)", o.OutputFormat)
	}
	return nil
}

func (o *Options) Run() error {
	ctx := context.Background()

	// Set namespace for API call
	ns := o.Namespace
	if o.AllNamespaces {
		ns = ""
	}

	needsNode := false
	needsSA := false
	needsPVC := false

	nodeMap := make(map[string]*corev1.Node)
	saMap := make(map[string]*corev1.ServiceAccount)
	pvcMap := make(map[string]*corev1.PersistentVolumeClaim)

	if strings.Contains(o.OutputFormat, ".node") {
		needsNode = true
	}

	if strings.Contains(o.OutputFormat, ".sa") || strings.Contains(o.OutputFormat, ".serviceAccount") {
		needsSA = true
	}

	if strings.Contains(o.OutputFormat, ".pvc") || strings.Contains(o.OutputFormat, ".pvcs") {
		needsPVC = true
	}

	// Get pods
	pods, err := o.Clientset.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	// Get nodes
	if needsNode {
		nodes, err := o.Clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list nodes: %w", err)
		}

		// Create node map for quick lookup
		for i := range nodes.Items {
			nodeMap[nodes.Items[i].Name] = &nodes.Items[i]
		}
	}

	if needsPVC {
		// Get all PVCs if needed
		allPVCs, err := o.Clientset.CoreV1().PersistentVolumeClaims(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list PVCs: %w", err)
		}

		// Create PVC map for quick lookup (namespace/name -> PVC)
		for i := range allPVCs.Items {
			key := allPVCs.Items[i].Namespace + "/" + allPVCs.Items[i].Name
			pvcMap[key] = &allPVCs.Items[i]
		}
	}

	if needsSA {
		// Get all ServiceAccounts if needed
		allSAs, err := o.Clientset.CoreV1().ServiceAccounts(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return fmt.Errorf("failed to list ServiceAccounts: %w", err)
		}

		// Create ServiceAccount map for quick lookup (namespace/name -> SA)
		for i := range allSAs.Items {
			key := allSAs.Items[i].Namespace + "/" + allSAs.Items[i].Name
			saMap[key] = &allSAs.Items[i]
		}
	}

	// Build pod with node information
	var podNodes []PodWithWider
	for i := range pods.Items {
		pod := &pods.Items[i]
		node := nodeMap[pod.Spec.NodeName]

		// Get ServiceAccount
		var sa *corev1.ServiceAccount
		if pod.Spec.ServiceAccountName != "" {
			saKey := pod.Namespace + "/" + pod.Spec.ServiceAccountName
			sa = saMap[saKey]
			// If not in map, try to fetch it directly
			if sa == nil {
				fetchedSA, err := o.Clientset.CoreV1().ServiceAccounts(pod.Namespace).Get(ctx, pod.Spec.ServiceAccountName, metav1.GetOptions{})
				if err == nil {
					sa = fetchedSA
				}
			}
		}

		// Get PVCs for this pod
		var podPVCs []*corev1.PersistentVolumeClaim
		for _, vol := range pod.Spec.Volumes {
			if vol.PersistentVolumeClaim != nil {
				pvcKey := pod.Namespace + "/" + vol.PersistentVolumeClaim.ClaimName
				if pvc, ok := pvcMap[pvcKey]; ok {
					podPVCs = append(podPVCs, pvc)
				} else {
					// If not in map, try to fetch it directly
					fetchedPVC, err := o.Clientset.CoreV1().PersistentVolumeClaims(pod.Namespace).Get(ctx, vol.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
					if err == nil {
						podPVCs = append(podPVCs, fetchedPVC)
					}
				}
			}
		}

		podNodes = append(podNodes, PodWithWider{
			Pod:            pod,
			Node:           node,
			ServiceAccount: sa,
			PVCs:           podPVCs,
		})
	}

	// Output
	if strings.HasPrefix(o.OutputFormat, "custom-columns=") {
		return o.printCustomColumns(podNodes)
	}

	return o.printDefault(podNodes)
}
