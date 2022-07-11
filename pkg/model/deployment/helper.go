package deployment

import (
	"encoding/json"
	"math/rand"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

type OAMWebServiceProperties struct {
	Labels          map[string]string                   `json:"labels,omitempty"`
	Annotations     map[string]string                   `json:"annotations,omitempty"`
	Image           string                              `json:"image,omitempty"`
	ImagePullPolicy string                              `json:"imagePullPolicy,omitempty"`
	Ports           []OAMWebServicePropertiesPort       `json:"ports,omitempty"`
	ExposeType      string                              `json:"exposeType,omitempty"`
	CMD             []string                            `json:"cmd,omitempty"`
	ENV             []OAMWebServicePropertiesEnv        `json:"env,omitempty"`
	CPU             string                              `json:"cpu,omitempty"`
	Memory          string                              `json:"memory,omitempty"`
	VolumeMounts    OAMWebServicePropertiesVolumeMounts `json:"volumeMounts,omitempty"`
}

type OAMWebServicePropertiesEnv struct {
	Name      string               `json:"name"`
	Value     string               `json:"value"`
	ValueFrom *corev1.EnvVarSource `json:"valueFrom,omitempty"`
}

func (o OAMWebServiceProperties) RawExtension() *runtime.RawExtension {
	raw, _ := json.Marshal(o)
	return &runtime.RawExtension{Raw: raw}
}

type OAMWebServicePropertiesPort struct {
	Port     int32  `json:"port,omitempty"`
	Name     string `json:"name,omitempty"`
	Protocol string `json:"protocol,omitempty"`
	Expose   bool   `json:"expose,omitempty"`
}

type OAMWebServicePropertiesVolumeMounts struct {
	PVC       []OAMWebServicePropertiesVolumeMount `json:"pvc,omitempty"`
	ConfigMap []OAMWebServicePropertiesVolumeMount `json:"configMap,omitempty"`
	Secret    []OAMWebServicePropertiesVolumeMount `json:"secret,omitempty"`
	EmptyDir  []OAMWebServicePropertiesVolumeMount `json:"emptyDir,omitempty"`
	HostPath  []OAMWebServicePropertiesVolumeMount `json:"hostPath,omitempty"`
}
type OAMWebServicePropertiesVolumeMount struct {
	Name        string `json:"name,omitempty"`
	MountPath   string `json:"mountPath,omitempty"`
	ClaimName   string `json:"claimName,omitempty"`
	CMName      string `json:"cmName,omitempty"`
	SecretName  string `json:"secretName,omitempty"`
	Medium      string `json:"medium,omitempty"` // when EmptyDir
	Path        string `json:"path,omitempty"`   // when HostPath
	DefaultMode int    `json:"defaultMode,omitempty"`

	Items []OAMWebServicePropertiesVolumeMountItem `json:"items,omitempty"`
}

type OAMWebServicePropertiesVolumeMountItem struct {
	Key  string `json:"key,omitempty"`
	Path string `json:"path,omitempty"`
	Mode int    `json:"mode,omitempty"`
}

// RandStringRunes generates a random string of letters and digits (lowercase)
func RandStringRunes(n int) string {
	// in ascii, a=97 b=98 c=99 d=100 ... z=122
	// for rand, a'=0 b'=1 c'=2 ... z'=25
	// so  i=i'+97 (a<i<z)
	const asciioffset = 97
	const azcount = 25
	b := make([]rune, n)
	for i := range b {
		b[i] = rune(rand.Intn(azcount) + asciioffset)
	}
	return string(b)
}

func Mergekvs(kvs map[string]string, into map[string]string) map[string]string {
	if into == nil {
		into = make(map[string]string)
	}
	for k, v := range kvs {
		into[k] = v
	}
	return into
}
