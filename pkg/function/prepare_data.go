package function

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	kanister "github.com/kanisterio/kanister/pkg"
	"github.com/kanisterio/kanister/pkg/format"
	"github.com/kanisterio/kanister/pkg/kube"
	"github.com/kanisterio/kanister/pkg/param"
)

const (
	defaultMountPoint         = "/mnt/prepare_data/%s"
	prepareDataJobPrefix      = "prepare-data-job-"
	PrepareDataNamespaceArg   = "namespace"
	PrepareDataImageArg       = "image"
	PrepareDataCommandArg     = "command"
	PrepareDataVolumes        = "volumes"
	PrepareDataServiceAccount = "serviceaccount"
)

func init() {
	kanister.Register(&prepareDataFunc{})
}

var _ kanister.Func = (*prepareDataFunc)(nil)

type prepareDataFunc struct{}

func (*prepareDataFunc) Name() string {
	return "PrepareData"
}

func getVolumes(tp param.TemplateParams) (map[string]string, error) {
	vols := make(map[string]string)
	var podsToPvcs map[string]map[string]string
	switch {
	case tp.Deployment != nil:
		podsToPvcs = tp.Deployment.PersistentVolumeClaims
	case tp.StatefulSet != nil:
		podsToPvcs = tp.StatefulSet.PersistentVolumeClaims
	default:
		return nil, errors.New("Failed to get volumes")
	}
	for _, podToPvcs := range podsToPvcs {
		for pvc := range podToPvcs {
			vols[pvc] = fmt.Sprintf(defaultMountPoint, pvc)
		}
	}
	if len(vols) == 0 {
		return nil, errors.New("No volumes found")
	}
	return vols, nil
}

func prepareData(ctx context.Context, cli kubernetes.Interface, namespace, serviceAccount, image string, vols map[string]string, command ...string) (map[string]interface{}, error) {
	// Validate volumes
	for pvc := range vols {
		if _, err := cli.CoreV1().PersistentVolumeClaims(namespace).Get(pvc, metav1.GetOptions{}); err != nil {
			return nil, errors.Wrapf(err, "Failed to retrieve PVC. Namespace %s, Name %s", namespace, pvc)
		}
	}
	pod, err := kube.CreatePod(ctx, cli, &kube.PodOptions{
		Namespace:          namespace,
		GenerateName:       prepareDataJobPrefix,
		Image:              image,
		Command:            command,
		Volumes:            vols,
		ServiceAccountName: serviceAccount,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create pod to run prepare data job")
	}
	defer func() {
		if err := kube.DeletePod(context.Background(), cli, pod); err != nil {
			log.Error("Failed to delete pod ", err.Error())
		}
	}()

	// Wait for pod completion
	if err := kube.WaitForPodCompletion(ctx, cli, pod.Namespace, pod.Name); err != nil {
		return nil, errors.Wrapf(err, "Failed while waiting for Pod %s to complete", pod.Name)
	}
	// Fetch logs from the pod
	logs, err := kube.GetPodLogs(ctx, cli, pod.Namespace, pod.Name)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to fetch logs from the pod")
	}
	format.Log(pod.Name, pod.Spec.Containers[0].Name, logs)

	out, err := parseLogAndCreateOutput(logs)
	return out, errors.Wrap(err, "Failed to parse phase output")
}

func (*prepareDataFunc) Exec(ctx context.Context, tp param.TemplateParams, args map[string]interface{}) (map[string]interface{}, error) {
	var namespace, image, serviceAccount string
	var command []string
	var vols map[string]string
	var err error
	if err = Arg(args, PrepareDataNamespaceArg, &namespace); err != nil {
		return nil, err
	}
	if err = Arg(args, PrepareDataImageArg, &image); err != nil {
		return nil, err
	}
	if err = Arg(args, PrepareDataCommandArg, &command); err != nil {
		return nil, err
	}
	if err = OptArg(args, PrepareDataVolumes, &vols, nil); err != nil {
		return nil, err
	}
	if err = OptArg(args, PrepareDataServiceAccount, &serviceAccount, ""); err != nil {
		return nil, err
	}
	cli, err := kube.NewClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create Kubernetes client")
	}
	if len(vols) == 0 {
		if vols, err = getVolumes(tp); err != nil {
			return nil, err
		}
	}
	return prepareData(ctx, cli, namespace, serviceAccount, image, vols, command...)
}

func (*prepareDataFunc) RequiredArgs() []string {
	return []string{PrepareDataNamespaceArg, PrepareDataImageArg, PrepareDataCommandArg}
}
