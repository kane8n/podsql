package main

import (
	"context"
	"fmt"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func RunPod(namespace, podName string, dbCommander DBCommander) (string, error) {
	clientset, _, err := newClientset()
	if err != nil {
		return "", fmt.Errorf("failed to create clientset: %w", err)
	}

	cmName := fmt.Sprintf("%s-cm", podName)

	// Define specifications to create pods
	podSpec := createRunPodSpec(podName, namespace, cmName, dbCommander)

	// create pods
	podsClient := clientset.CoreV1().Pods(namespace)
	pod, err := podsClient.Create(context.Background(), podSpec, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create pod: %w", err)
	}

	// Create ConfigMap to hold queries
	// Because the -Q option of sqlcmd does not allow queries over 1K to be executed, use ConfigMap to transfer the sql file to the pod and execute it with the -i option.
	configMap := createConfigMapSpec(cmName, namespace, pod, map[string]string{"query.sql": dbCommander.Query()})
	_, err = clientset.CoreV1().ConfigMaps(namespace).Create(context.Background(), configMap, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create configmap: %w", err)
	}

	// Create Secret for DB_USER and DB_PASSWORD with argument values
	secret := createBasicAuthSecretSpec(fmt.Sprintf("%s-secret", podName), namespace, dbCommander.ConnectInfo(), pod)
	_, err = clientset.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		return "", fmt.Errorf("failed to create secret: %w", err)
	}

	if err = waitForPodRunning(context.Background(), podsClient, podName); err != nil {
		return "", err
	}

	req := podsClient.GetLogs(podName, &corev1.PodLogOptions{
		Follow: true,
	})

	podLogs, err := req.Stream(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get pod logs: %w", err)
	}
	defer podLogs.Close()

	var logs []byte
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := podLogs.Read(buf)
			if err != nil {
				if err == io.EOF {
					break
				}

				fmt.Printf("failed to read pod logs: %v\n", err)
				break
			}
			logs = append(logs, buf[:n]...)
		}
	}()

	waitCh := make(chan struct{})
	go func() {
		for {
			select {
			case <-time.After(1 * time.Second): // Adjust the interval as needed
				pod, err := podsClient.Get(context.Background(), podName, metav1.GetOptions{})
				if err != nil {
					fmt.Printf("Error getting Pod: %v\n", err)
					continue
				}
				if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
					close(waitCh)
					return
				}
			}
		}
	}()

	// Wait for the pod to terminate
	<-waitCh

	// Delete the pod
	if err := deletePod(podsClient, podName); err != nil {
		return "", fmt.Errorf("failed to delete pod: %w", err)
	}
	return string(logs), nil
}

func createRunPodSpec(podName, namespace, cmName string, dbCommander DBCommander) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Volumes: []corev1.Volume{
				{
					Name: "query-volume",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: cmName,
							},
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:         dbCommander.CommandType().String(),
					Image:        dbCommander.ContainerImage(),
					VolumeMounts: []corev1.VolumeMount{{Name: "query-volume", MountPath: "/sql"}},
					Env:          append(generateSecretEnvVars(podName, dbCommander), corev1.EnvVar{Name: "TZ", Value: "Asia/Tokyo"}), // FIXME
					Command:      []string{"/bin/sh", "-c", dbCommander.Command()},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
}

func createConfigMapSpec(cmName, namespace string, pod *corev1.Pod, data map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(pod, corev1.SchemeGroupVersion.WithKind("Pod")),
			},
		},
		Data: data,
	}
}

func createConfigMap(clientset *kubernetes.Clientset, podName, namespace string, data map[string]string) (*corev1.ConfigMap, error) {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-cm", podName),
			Namespace: namespace,
		},
		Data: data,
	}
	cm, err := clientset.CoreV1().ConfigMaps(namespace).Create(context.Background(), configMap, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create configmap: %w", err)
	}
	return cm, nil
}
