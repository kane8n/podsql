package main

import (
	"context"
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/util/term"
)

func ExecPod(namespace, podName string, dbCommander DBCommander) error {
	clientset, config, err := newClientset()
	if err != nil {
		return fmt.Errorf("failed to create clientset: %w", err)
	}

	// Define specifications to create pods
	podSpec := createExecPodSpec(podName, namespace, dbCommander)

	// create pods
	podsClient := clientset.CoreV1().Pods(namespace)
	pod, err := podsClient.Create(context.Background(), podSpec, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create pod: %w", err)
	}

	// Create Secret for DB_USER and DB_PASSWORD with argument values
	secret := createBasicAuthSecretSpec(fmt.Sprintf("%s-secret", podName), namespace, dbCommander.ConnectInfo(), pod)
	_, err = clientset.CoreV1().Secrets(namespace).Create(context.Background(), secret, metav1.CreateOptions{})
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	if err = waitForPodRunning(context.Background(), podsClient, podName); err != nil {
		return err
	}

	req := clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: dbCommander.CommandType().String(),
			Command:   []string{"/bin/sh", "-c", dbCommander.InteractiveCommand()},
			Stdin:     true,
			Stdout:    true,
			Stderr:    false,
			TTY:       true,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	tty := term.TTY{
		Out: os.Stdout,
		In:  os.Stdin,
		Raw: true,
	}

	err = tty.Safe(func() error {
		return exec.StreamWithContext(context.Background(), remotecommand.StreamOptions{
			Stdin:             tty.In,
			Stdout:            tty.Out,
			Stderr:            nil,
			Tty:               true,
			TerminalSizeQueue: tty.MonitorSize(tty.GetSize()),
		})
	})
	if err != nil {
		if delerr := deletePod(podsClient, podName); delerr != nil {
			return fmt.Errorf("failed to execute command: %w, failed to delete pod: %w", err, delerr)
		}
		return fmt.Errorf("failed to execute command: %w", err)
	}

	// Delete the pod
	if err := deletePod(podsClient, podName); err != nil {
		return err
	}
	return nil
}

func createExecPodSpec(podName, namespace string, dbCommander DBCommander) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    dbCommander.CommandType().String(),
					Image:   dbCommander.ContainerImage(),
					Env:     append(generateSecretEnvVars(podName, dbCommander), corev1.EnvVar{Name: "TZ", Value: "Asia/Tokyo"}), // FIXME
					Command: []string{"/bin/sh", "-c", "tail -f /dev/null"},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
}
