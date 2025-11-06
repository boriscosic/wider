package main

import (
	"reflect"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected string
	}{
		{
			name:     "30 seconds",
			duration: 30 * time.Second,
			expected: "30s",
		},
		{
			name:     "5 minutes",
			duration: 5 * time.Minute,
			expected: "5m",
		},
		{
			name:     "2 hours",
			duration: 2 * time.Hour,
			expected: "2h",
		},
		{
			name:     "29 hours",
			duration: 29 * time.Hour,
			expected: "29h",
		},
		{
			name:     "47 hours",
			duration: 47 * time.Hour,
			expected: "47h",
		},
		{
			name:     "2 days",
			duration: 48 * time.Hour,
			expected: "2d",
		},
		{
			name:     "5 days",
			duration: 5 * 24 * time.Hour,
			expected: "5d",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creationTime := metav1.NewTime(time.Now().Add(-tt.duration))
			result := formatAge(creationTime)
			if result != tt.expected {
				t.Errorf("formatAge() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"metadata", "Metadata"},
		{"name", "Name"},
		{"status", "Status"},
		{"", ""},
		{"a", "A"},
		{"ABC", "ABC"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := capitalizeFirst(tt.input)
			if result != tt.expected {
				t.Errorf("capitalizeFirst(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestSplitPath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name:     "simple path",
			path:     "pod.metadata.name",
			expected: []string{"pod", "metadata", "name"},
		},
		{
			name:     "path with escaped dot",
			path:     "node.metadata.labels.kubernetes\\.io/os",
			expected: []string{"node", "metadata", "labels", "kubernetes.io/os"},
		},
		{
			name:     "multiple escaped dots",
			path:     "a\\.b.c\\.d.e",
			expected: []string{"a.b", "c.d", "e"},
		},
		{
			name:     "leading dot",
			path:     ".pod.metadata.name",
			expected: []string{"pod", "metadata", "name"},
		},
		{
			name:     "trailing dot",
			path:     "pod.metadata.",
			expected: []string{"pod", "metadata"},
		},
		{
			name:     "empty string",
			path:     "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitPath(tt.path)
			if len(result) != len(tt.expected) {
				t.Errorf("splitPath(%q) length = %v, want %v", tt.path, len(result), len(tt.expected))
				return
			}
			for i := range result {
				if result[i] != tt.expected[i] {
					t.Errorf("splitPath(%q)[%d] = %v, want %v", tt.path, i, result[i], tt.expected[i])
				}
			}
		})
	}
}

func TestGetValueByPath(t *testing.T) {
	// Create test pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pod",
			Namespace: "default",
			Labels: map[string]string{
				"app": "myapp",
			},
			OwnerReferences: []metav1.OwnerReference{
				{
					Kind: "ReplicaSet",
					Name: "test-rs",
				},
			},
		},
		Spec: corev1.PodSpec{
			NodeName:           "node1",
			ServiceAccountName: "default",
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}

	// Create test node
	node := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "node1",
			Labels: map[string]string{
				"kubernetes.io/os": "linux",
			},
		},
		Status: corev1.NodeStatus{
			NodeInfo: corev1.NodeSystemInfo{
				OperatingSystem: "linux",
			},
		},
	}

	// Create test ServiceAccount
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "default",
		},
	}

	// Create test PVC
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "default",
		},
	}

	pn := PodWithWider{
		Pod:            pod,
		Node:           node,
		ServiceAccount: sa,
		PVCs:           []*corev1.PersistentVolumeClaim{pvc},
	}

	tests := []struct {
		name     string
		path     string
		expected string
		wantErr  bool
	}{
		{
			name:     "pod name",
			path:     ".pod.metadata.name",
			expected: "test-pod",
			wantErr:  false,
		},
		{
			name:     "pod namespace",
			path:     "pod.metadata.namespace",
			expected: "default",
			wantErr:  false,
		},
		{
			name:     "pod label",
			path:     "pod.metadata.labels.app",
			expected: "myapp",
			wantErr:  false,
		},
		{
			name:     "node name",
			path:     ".node.metadata.name",
			expected: "node1",
			wantErr:  false,
		},
		{
			name:     "node label with escaped dot",
			path:     ".node.metadata.labels.kubernetes\\.io/os",
			expected: "linux",
			wantErr:  false,
		},
		{
			name:     "node OS",
			path:     ".node.status.nodeInfo.operatingSystem",
			expected: "linux",
			wantErr:  false,
		},
		{
			name:     "service account name",
			path:     ".serviceAccount.metadata.name",
			expected: "default",
			wantErr:  false,
		},
		{
			name:     "service account via sa alias",
			path:     ".sa.metadata.name",
			expected: "default",
			wantErr:  false,
		},
		{
			name:     "pvcs list",
			path:     ".pvcs",
			expected: "test-pvc",
			wantErr:  false,
		},
		{
			name:     "pod status phase",
			path:     ".pod.status.phase",
			expected: "Running",
			wantErr:  false,
		},
		{
			name:     "invalid path start",
			path:     ".invalid.metadata.name",
			expected: "",
			wantErr:  true,
		},
		{
			name:     "nonexistent field",
			path:     ".pod.metadata.nonexistent",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := getValueByPath(pn, tt.path)
			if tt.wantErr {
				if err == nil {
					t.Errorf("getValueByPath(%q) expected error but got none", tt.path)
				}
				return
			}
			if err != nil {
				t.Errorf("getValueByPath(%q) unexpected error: %v", tt.path, err)
				return
			}
			if result != tt.expected {
				t.Errorf("getValueByPath(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetValueByPath_NilNode(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
	}

	pn := PodWithWider{
		Pod:  pod,
		Node: nil, // Node not assigned yet
	}

	result, err := getValueByPath(pn, ".node.metadata.name")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "<none>" {
		t.Errorf("expected <none> for nil node, got %v", result)
	}
}

func TestGetValueByPath_NilServiceAccount(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
	}

	pn := PodWithWider{
		Pod:            pod,
		ServiceAccount: nil,
	}

	result, err := getValueByPath(pn, ".sa.metadata.name")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "<none>" {
		t.Errorf("expected <none> for nil service account, got %v", result)
	}
}

func TestGetValueByPath_EmptyPVCs(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
	}

	pn := PodWithWider{
		Pod:  pod,
		PVCs: nil,
	}

	result, err := getValueByPath(pn, ".pvcs")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "<none>" {
		t.Errorf("expected <none> for empty PVCs, got %v", result)
	}
}

func TestFindFieldByJSONTag(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
		},
	}

	val := findFieldByJSONTag(reflect.ValueOf(*pod), "metadata")
	if !val.IsValid() {
		t.Error("expected to find 'metadata' field by JSON tag")
	}

	val = findFieldByJSONTag(reflect.ValueOf(*pod), "nonexistent")
	if val.IsValid() {
		t.Error("expected not to find 'nonexistent' field")
	}
}

func TestOptionsValidate(t *testing.T) {
	tests := []struct {
		name         string
		outputFormat string
		wantErr      bool
	}{
		{
			name:         "empty format",
			outputFormat: "",
			wantErr:      false,
		},
		{
			name:         "valid custom-columns",
			outputFormat: "custom-columns=NAME:.pod.metadata.name",
			wantErr:      false,
		},
		{
			name:         "invalid format",
			outputFormat: "json",
			wantErr:      true,
		},
		{
			name:         "invalid format yaml",
			outputFormat: "yaml",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &Options{
				OutputFormat: tt.outputFormat,
			}
			err := opts.Validate()
			if tt.wantErr && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
