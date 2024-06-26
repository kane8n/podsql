package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type ImagePullBackOffError struct {
	Message string
	Reason  string
}

func (e *ImagePullBackOffError) Error() string {
	return fmt.Sprintf("reason: %s, message: %s", e.Reason, e.Message)
}

func CreatePodName(prefix string) (string, error) {
	currentUser, err := user.Current()
	formatedCurrentUserName := strings.ReplaceAll(currentUser.Username, "_", "-")
	formatedCurrentUserName = strings.ReplaceAll(formatedCurrentUserName, ".", "")
	if err != nil {
		return "", fmt.Errorf("failed to get current user: %w", err)
	}
	tz, err := time.LoadLocation("Asia/Tokyo") // FIXME
	if err != nil {
		return "", fmt.Errorf("failed to load location: %w", err)
	}
	return fmt.Sprintf(
		"%s-%s-%s",
		prefix,
		time.Now().In(tz).Format("20060102-150405"),
		formatedCurrentUserName), nil
}

func createBasicAuthSecretSpec(secretName, namespace string, connectInfo ConnectInfo, pod *corev1.Pod) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(pod, corev1.SchemeGroupVersion.WithKind("Pod")),
			},
		},
		Type: corev1.SecretTypeBasicAuth,
		StringData: map[string]string{
			"username": connectInfo.User,
			"password": connectInfo.Password,
		},
	}
}

func newClientset() (*kubernetes.Clientset, *rest.Config, error) {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		kubeconfig = filepath.Join(homedir.HomeDir(), ".kube", "config")
	}

	// Setup Kubernetes client
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build config from flags: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)

	return clientset, config, err
}

func waitForPodRunning(ctx context.Context, podsClient v1.PodInterface, podName string) error {
	if err := wait.PollUntilContextTimeout(ctx, 1*time.Second, 1*time.Minute, true, func(context.Context) (bool, error) {
		pod, err := podsClient.Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return false, fmt.Errorf("failed to get pod: %w", err)
		}

		if pod.Status.Phase == corev1.PodRunning || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			return true, nil
		}

		for _, st := range pod.Status.ContainerStatuses {
			if st.State.Waiting != nil && st.State.Waiting.Reason == "ImagePullBackOff" {
				return false, &ImagePullBackOffError{Reason: st.State.Waiting.Reason, Message: st.State.Waiting.Message}
			}
		}
		return false, nil
	}); err != nil {
		switch e := err.(type) {
		case *ImagePullBackOffError:
			// Delete the pod if the image pull backoff occurs
			err := fmt.Errorf("pod running failed. %w", e)
			deletePolicy := metav1.DeletePropagationForeground
			if derr := podsClient.Delete(context.Background(), podName, metav1.DeleteOptions{
				PropagationPolicy: &deletePolicy,
			}); derr != nil {
				return errors.Join(err, fmt.Errorf("failed to delete pod: %w", derr))
			}
			return err
		}
		return fmt.Errorf("failed to wait for pod running: %w", err)
	}
	return nil
}

func deletePod(podsClient v1.PodInterface, podName string) error {
	// Delete the pod
	deletePolicy := metav1.DeletePropagationForeground
	if err := podsClient.Delete(context.Background(), podName, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	}); err != nil {
		return fmt.Errorf("failed to delete pod: %w", err)
	}
	return nil
}

func generateSecretEnvVars(podName string, dbCommander DBCommander) []corev1.EnvVar {
	secretName := fmt.Sprintf("%s-secret", podName)
	envVars := []corev1.EnvVar{}
	for k, v := range dbCommander.SecretEnvKV() {
		envVars = append(envVars, generateEnvVar(secretName, k, v))
	}
	return envVars
}

func generateEnvVar(secretName, name, key string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
				Key: key,
			},
		},
	}
}
