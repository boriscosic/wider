package main

import (
	"encoding/json"
	"fmt"
	corev1 "k8s.io/api/core/v1"
	"os"
	"sigs.k8s.io/yaml"
	"strings"
	"text/tabwriter"
)

func (o *Options) printJSON(podNodes []PodWithWider) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(podNodes)
}

func (o *Options) printYAML(podNodes []PodWithWider) error {
	data, err := yaml.Marshal(podNodes)
	if err != nil {
		return fmt.Errorf("failed to marshal to YAML: %w", err)
	}
	fmt.Println(string(data))
	return nil
}

func (o *Options) printCustomColumns(podNodes []PodWithWider) error {
	// Parse custom-columns format
	columnsStr := strings.TrimPrefix(o.OutputFormat, "custom-columns=")
	columnDefs := strings.Split(columnsStr, ",")

	var headers []string
	var paths []string

	for _, def := range columnDefs {
		parts := strings.SplitN(def, ":", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid custom-columns format: %s", def)
		}
		headers = append(headers, parts[0])
		paths = append(paths, parts[1])
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	// Print headers
	fmt.Fprintln(w, strings.Join(headers, "\t"))

	// Print rows
	for _, pn := range podNodes {
		var values []string
		for _, path := range paths {
			val, err := getValueByPath(pn, path)
			if err != nil {
				values = append(values, "<none>")
			} else {
				values = append(values, val)
			}
		}
		fmt.Fprintln(w, strings.Join(values, "\t"))
	}

	return nil
}

func (o *Options) printDefault(podNodes []PodWithWider) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	defer w.Flush()

	if o.AllNamespaces {
		fmt.Fprintln(w, "NAMESPACE\tNAME\tREADY\tSTATUS\tRESTARTS\tAGE\tIP\tNODE")
	} else {
		fmt.Fprintln(w, "NAME\tREADY\tSTATUS\tRESTARTS\tAGE\tIP\tNODE")
	}

	for _, pn := range podNodes {
		pod := pn.Pod

		// Calculate READY (ready/total containers)
		totalContainers := len(pod.Spec.Containers)
		readyContainers := 0
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.Ready {
				readyContainers++
			}
		}
		ready := fmt.Sprintf("%d/%d", readyContainers, totalContainers)

		// Get STATUS
		status := string(pod.Status.Phase)
		if pod.DeletionTimestamp != nil {
			status = "Terminating"
		}

		// Calculate RESTARTS
		restarts := 0
		for _, cs := range pod.Status.ContainerStatuses {
			restarts += int(cs.RestartCount)
		}

		// Calculate AGE
		age := formatAge(pod.CreationTimestamp)

		// Get NODE info
		nodeName := pod.Spec.NodeName
		nodeIP := ""
		if pn.Node != nil {
			for _, addr := range pn.Node.Status.Addresses {
				if addr.Type == corev1.NodeInternalIP {
					nodeIP = addr.Address
					break
				}
			}
		}

		if o.AllNamespaces {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
				pod.Namespace,
				pod.Name,
				ready,
				status,
				restarts,
				age,
				nodeIP,
				nodeName)
		} else {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
				pod.Name,
				ready,
				status,
				restarts,
				age,
				nodeIP,
				nodeName)
		}
	}

	return nil
}
