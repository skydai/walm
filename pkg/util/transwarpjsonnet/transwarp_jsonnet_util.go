package transwarpjsonnet

import (
	k8sModel "WarpCloud/walm/pkg/models/k8s"
	"WarpCloud/walm/pkg/util/transwarpjsonnet/memkv"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"github.com/Masterminds/sprig"
	yaml2 "github.com/ghodss/yaml"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
	"io"
	"io/ioutil"
	"k8s.io/klog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"text/template/parse"

	"github.com/google/go-jsonnet"
	jsonnetAst "github.com/google/go-jsonnet/ast"
	"gopkg.in/yaml.v2"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

type JsonnetConfig struct {
	FileName string
}

type volumeMount struct {
	Name      string `json:"name"`
	MountPath string `json:"mountPath"`
	SubPath   string `json:"subPath"`
}

type volume struct {
	Key  string `json:"key"`
	Path string `json:"path"`
	Mode int    `json:"mode"`
}

type configFilesResult struct {
	Name             string            `json:"name"`
	ConfigMapDataMap map[string]string `json:"configMapsData"`
	VolumeList       []volume          `json:"volumes"`
	VolumeMount      volumeMount       `json:"volumeMount"`
	VolumeMountList  []volumeMount     `json:"volumeMounts"`
	Md5Checksum      string            `json:"md5Checksum"`
}

type configFilesVolConfig struct {
	ConfigMapKey        string `json:"configMapKey"`
	FileLocation        string `json:"fileLocation"`
	FileData            string `json:"fileData"`
	VolConfigMapSubPath string `json:"volConfigMapSubPath"`
	VolConfigMapMode    int    `json:"volConfigMapMode"`
}

func loadConfigFiles(name string, mountPath string, volConfigsVal interface{}, mainPath string) (interface{}, error) {
	var allContent string
	volConfigList := make([]configFilesVolConfig, 0)
	volConfigsBytes, err := json.Marshal(volConfigsVal)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(volConfigsBytes, &volConfigList)
	if err != nil {
		return nil, err
	}
	resObj := configFilesResult{
		Name:        name,
		Md5Checksum: "",
		VolumeMountList: []volumeMount{
			{
				Name:      name,
				MountPath: mountPath,
			},
		},
		VolumeList:       make([]volume, 0),
		ConfigMapDataMap: make(map[string]string, 0),
	}
	for _, volConfig := range volConfigList {
		var content string
		if volConfig.FileData != "" {
			content = volConfig.FileData
		} else {
			location := volConfig.FileLocation
			if mainPath != "" {
				location = filepath.Join(filepath.Dir(mainPath), location)
			}
			file, err := os.Open(location)
			if err != nil {
				return nil, err
			}
			fileContent, err := ioutil.ReadAll(file)
			if err != nil {
				file.Close()
				return nil, err
			}
			file.Close()
			content = string(fileContent[:])
		}

		resObj.ConfigMapDataMap[volConfig.ConfigMapKey] = content
		allContent += content

		volumeConfig := volume{
			Key:  volConfig.ConfigMapKey,
			Path: volConfig.VolConfigMapSubPath,
			Mode: 420,
		}
		if volConfig.VolConfigMapMode != 0 {
			volumeConfig.Mode = volConfig.VolConfigMapMode
		}
		resObj.VolumeList = append(resObj.VolumeList, volumeConfig)
	}
	md5Data := md5.Sum([]byte(allContent))
	resObj.Md5Checksum = string(md5Data[:])
	data, err := json.Marshal(resObj)
	if err != nil {
		return nil, err
	}
	var res interface{}
	err = k8syaml.NewYAMLToJSONDecoder(bytes.NewReader(data)).Decode(&res)
	if err != nil {
		return nil, err
	}
	return res, err
}

func gotmplRender(tmplContent string, context interface{}, returnType string) (interface{}, error) {
	var doc bytes.Buffer
	t, err := template.New("jsonnetGoTmpl").Parse(tmplContent)
	if err != nil {
		return "", err
	}
	err = t.Execute(&doc, context)
	if err != nil {
		return "", err
	}
	if returnType == "json" {
		var res interface{}
		err = k8syaml.NewYAMLToJSONDecoder(bytes.NewReader([]byte(doc.String()[:]))).Decode(&res)
		if err != nil {
			return nil, err
		}
		return res, nil
	} else {
		return doc.String(), nil
	}
}

func RegisterNativeFuncs(vm *jsonnet.VM) {
	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "parseYaml",
		Params: []jsonnetAst.Identifier{"yaml"},
		Func: func(args []interface{}) (res interface{}, err error) {
			ret := []interface{}{}
			data := []byte(args[0].(string))
			d := k8syaml.NewYAMLToJSONDecoder(bytes.NewReader(data))
			for {
				var doc interface{}
				if err := d.Decode(&doc); err != nil {
					if err == io.EOF {
						break
					}
					return nil, err
				}
				ret = append(ret, doc)
			}
			return ret, nil
		},
	})

	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "loadConfigFiles",
		Params: []jsonnetAst.Identifier{"name", "mountPath", "volConfigs", "mainPath"},
		Func: func(args []interface{}) (res interface{}, err error) {
			return loadConfigFiles(args[0].(string), args[1].(string), args[2].(interface{}), args[3].(string))
		},
	})

	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "gotmplRender",
		Params: []jsonnetAst.Identifier{"tmplContent", "context", "returnType"},
		Func: func(args []interface{}) (res interface{}, err error) {
			return gotmplRender(args[0].(string), args[1].(interface{}), args[2].(string))
		},
	})

	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "renderConfdFiles",
		Params: []jsonnetAst.Identifier{"name", "confdKV", "confdFilesConfig", "mainPath"},
		Func: func(args []interface{}) (res interface{}, err error) {
			return renderConfdFiles(args[0].(string), args[1].(interface{}), args[2].(interface{}), args[3].(string))
		},
	})
}

func renderMainJsonnetFile(templateFiles map[string]string, configValues map[string]interface{}) (jsonStr string, err error) {
	tmpdir, err := ioutil.TempDir("", "jsonnet")
	if err != nil {
		klog.Errorf("create tempdir error %v", err)
		return "", err
	}
	defer os.RemoveAll(tmpdir)

	for filename, content := range templateFiles {
		tmpfn := filepath.Join(tmpdir, filename)
		os.MkdirAll(filepath.Dir(tmpfn), 0755)
		if err := ioutil.WriteFile(tmpfn, []byte(content[:]), 0666); err != nil {
			klog.Errorf("write to tempdir error %v", err)
		}
	}

	mainJsonFileName, err := getMainJsonnetFile(templateFiles)
	if err != nil {
		klog.Errorf("failed to get main jsonnet file : %s", err.Error())
		return "", err
	}

	tlaValue, err := json.Marshal(configValues)
	if err != nil {
		klog.Errorf("failed to marshal config values : %s", err.Error())
		return "", err
	}

	jsonStr, err = parseTemplateWithTLAString(filepath.ToSlash(filepath.Join(tmpdir, mainJsonFileName)), "config", string(tlaValue))
	if err != nil {
		klog.Errorf("failed to parse main jsonnet template file : %s", err.Error())
		return "", err
	}
	return
}

func BuildNotRenderedFileName(fileName string) (notRenderFileName string) {
	notRenderFileName = path.Join(path.Dir(fileName), path.Base(fileName)+TranswarpJsonetFileSuffix)
	return
}

func buildKubeResourcesByJsonStr(jsonStr string, labels map[string]string) (resources map[string][]byte, err error) {
	// key: resource.json, value: resource template(map)
	resourcesMap := make(map[string]map[string]interface{})
	err = json.Unmarshal([]byte(jsonStr), &resourcesMap)
	if err != nil {
		klog.Errorf("failed to unmarshal json string : %s", err.Error())
		return nil, err
	}

	// for pod getenv
	/*
	envMap := make(map[string]string)
	for _, resource := range resourcesMap {
		if  resource["kind"] == string(k8sModel.StatefulSetKind)  || resource["kind"] == string(k8sModel.DeploymentKind) {
			data, _ := json.Marshal(resource)

			envMapArray := gjson.GetBytes(data, "spec.template.spec.containers.#.env").Array()
			if len(envMapArray) == 0 {
				continue
			}

			for _, tmpEnvMap := range envMapArray {
				envs := tmpEnvMap.Array()
				for _, env := range envs {
					tmpValue := gjson.Get(env.String(), "value").String()
					tmpName := gjson.Get(env.String(), "name").String()
					if tmpValue != "" {
						envMap[tmpName] = tmpValue
					}
				}
			}

		}
	}
	*/

	resources = map[string][]byte{}
	for fileName, resource := range resourcesMap {
		// render with confd
		if  resource["kind"] == string(k8sModel.ConfigMapKind) {
			data := resource["data"].(map[string]interface{})
			newData, err := renderDataWithConfd(data)
			if err != nil {
				klog.Errorf("failed to render configmap template with confd")
				return nil, err
			}
			resource["data"] = newData
		}

		// set labels with each k8s resource
		resourceBytes, err := yaml.Marshal(resource)
		if err != nil {
			return nil, err
		}
		resourceBytes, err = yaml2.YAMLToJSON(resourceBytes)
		if err != nil {
			return nil, err
		}
		resourceStr := string(resourceBytes)
		for k, v := range labels {
			path := "metadata.labels." + k
			resourceStr, err = sjson.Set(resourceStr, path, v)
			if err != nil {
				klog.Errorf("failed to set %s to %s of resource %s: %s", v, path, fileName, err.Error())
				return nil, err
			}
			resourceKind := gjson.Get(resourceStr, "kind").String()
			if resourceKind == string(k8sModel.StatefulSetKind) || resourceKind == string(k8sModel.DeploymentKind){
				templatePath := "spec.template.metadata.labels." + k
				resourceStr, err = sjson.Set(resourceStr, templatePath, v)
				if err != nil {
					klog.Errorf("failed to set %s to %s of resource %s: %s", v, path, fileName, err.Error())
					return nil, err
				}
			}
		}
		resourceBytes, err = yaml2.JSONToYAML([]byte(resourceStr))

		if err != nil {
			klog.Errorf("failed to marshal resource to yaml bytes : %s", err.Error())
			return nil, err
		}
		resources[fileName] = resourceBytes
	}

	return
}

func getMainJsonnetFile(templateFiles map[string]string) (string, error) {
	for fileName := range templateFiles {
		if strings.HasSuffix(fileName, "main.jsonnet") {
			return fileName, nil
		}
	}
	return "", fmt.Errorf("failed to find main jsonnet file")
}

// parseTemplateWithTLAString parse the templates by specifying values of Top-Level Arguments (TLA)
// The TLAs comes from external json string.
func parseTemplateWithTLAString(templatePath string, tlaVar string, tlaValue string) (string, error) {
	vm := jsonnet.MakeVM()
	RegisterNativeFuncs(vm)
	if tlaVar != "" {
		vm.TLACode(tlaVar, tlaValue)
	}
	jsonnetBytes, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return "", err
	}

	output, err := vm.EvaluateSnippet(templatePath, string(jsonnetBytes))
	if err != nil {
		klog.Errorf("failed to parse template %s, %s=%s, error: %+v", templatePath, tlaVar, tlaValue, err)
		return "", err
	}
	return output, nil
}

func renderDataWithConfd(data map[string]interface{}) (map[string]interface{}, error) {
	tmplFiles := map[string]string{}
	renderedFiles := map[string]interface{}{}
	confdKV := make(map[string]string)
	var err error
	for file, fileData := range data {
		renderedFiles[file] = fileData
		if strings.HasSuffix(file, ".tmpl") || strings.HasSuffix(file, ".raw") {
			tmplFiles[file] = fmt.Sprintf("%v", fileData)
		} else if strings.HasSuffix(file, ".conf") {
			confdKV, err = getConfdKV(fileData, []string{"/"})
			if err != nil {
				klog.Errorf("failed to get confd kv from confd.conf: %s", err.Error())
				return nil, err
			}
		}
	}
	tmplRenderedFiles := map[string]interface{}{}
	for k, v := range tmplFiles {
		str, err := renderFileWithCfd(k, v, confdKV)
		if err != nil {
			if strings.Contains(err.Error(),"function \"getenv\" not defined") {
				tmplRenderedFiles[k] = v
				continue
			}
			klog.Errorf("failed to render resource file %s %v: %s", k, v, err.Error())
			return nil, err
		}
		renderedFiles[k] = str
	}

	return renderedFiles, nil
}

func renderFileWithCfd(filename string, data string, confdkv interface{}) (string, error) {
	var t *template.Template

	store := memkv.New()
	vars := make(map[string]string)
	yamlMap := make(map[string]interface{})
	byteKV, _ := json.Marshal(confdkv)
	err := json.Unmarshal(byteKV, &yamlMap)

	err = nodeWalk(yamlMap, "", vars)
	if err != nil {
		return "", err
	}

	for k, v := range vars {
		store.Set(path.Join("/", k), v)
	}

	t, err = template.New(filename).
		Funcs(sprig.TxtFuncMap()).
		Funcs(newFuncMap()).
		//Funcs(userDefinedFuncMap(envMap)).
		Funcs(store.FuncMap).Parse(data)
	if err != nil {
		return "", err
	}

	matchNames := make(map[string]struct{})
	matchNames["getenv"] = struct{}{}

	for idx, node := range t.Tree.Root.Nodes {
		if matchNode(node, matchNames) {
			textTmpl, err := template.New(filename).Funcs(sprig.TxtFuncMap()).Funcs(newFuncMap()).Funcs(store.FuncMap).Delims("{{{{{{", "}}}}}}").Parse(node.String())
			if err != nil {
				panic(err)
			}
			newNode := textTmpl.Root.Nodes[0]
			// newNode must be text node
			textNode := newNode.(*parse.TextNode)
			// assign node position to newNode
			textNode.Pos = node.Position()
			//t := reflect.ValueOf(&node).MethodByName("tree").Call([]reflect.Value{})
			t.Tree.Root.Nodes[idx] = textNode
		}
	}

	var fileTpl bytes.Buffer
	err = t.Execute(&fileTpl, confdkv)
	if err != nil {
		return "", err
	}
	return fileTpl.String(), nil
}

func matchActionNode(actionNode *parse.ActionNode, funcNames map[string]struct{}) bool {
	if actionNode == nil {
		return false
	}
	for _, cmdNode := range actionNode.Pipe.Cmds {
		if matchCmdNode(cmdNode, funcNames) {
			return true
		}
	}
	return false
}

func matchPipeNode(pipeNode *parse.PipeNode, funcNames map[string]struct{}) bool {
	if pipeNode == nil {
		return false
	}
	for _, cmdNode := range pipeNode.Cmds {
		if matchCmdNode(cmdNode, funcNames) {
			return true
		}
	}
	return false
}

func matchCmdNode(cmdNode *parse.CommandNode, funcNames map[string]struct{}) bool {
	if cmdNode == nil {
		return false
	}
	// check the first argument of command node
	if _, exists := funcNames[cmdNode.Args[0].String()]; exists {
		return true
	}
	// recursively check other arguments
	if len(cmdNode.Args) > 1 {
		for _, argNode := range cmdNode.Args[1:] {
			if matchNode(argNode, funcNames) {
				return true
			}
		}
	}
	return false
}

// used by if, range, with
func matchBranchNode(branchNode *parse.BranchNode, funcNames map[string]struct{}) bool {
	if branchNode == nil {
		return false
	}
	if matchPipeNode(branchNode.Pipe, funcNames) {
		return true
	}
	if matchListNode(branchNode.List, funcNames) {
		return true
	}
	if matchListNode(branchNode.ElseList, funcNames) {
		return true
	}
	return false
}

func matchListNode(listNode *parse.ListNode, funcNames map[string]struct{}) bool {
	if listNode == nil {
		return false
	}
	for _, node := range listNode.Nodes {
		if matchNode(node, funcNames) {
			return true
		}
	}
	return false
}

func matchIfNode(ifNode *parse.IfNode, funcNames map[string]struct{}) bool {
	if ifNode == nil {
		return false
	}
	return matchBranchNode(&ifNode.BranchNode, funcNames)
}

func matchRangeNode(rangeNode *parse.RangeNode, funcNames map[string]struct{}) bool {
	if rangeNode == nil {
		return false
	}
	return matchBranchNode(&rangeNode.BranchNode, funcNames)
}

func matchWithNode(withNode *parse.WithNode, funcNames map[string]struct{}) bool {
	if withNode == nil {
		return false
	}
	return matchBranchNode(&withNode.BranchNode, funcNames)
}

func matchTemplateNode(templateNode *parse.TemplateNode, funcNames map[string]struct{}) bool {
	if templateNode == nil {
		return false
	}
	return matchPipeNode(templateNode.Pipe, funcNames)
}

// traverse node to match given function name
func matchNode(node parse.Node, funcNames map[string]struct{}) bool {
	switch node.Type() {
	case parse.NodeAction:
		actionNode := node.(*parse.ActionNode)
		return matchActionNode(actionNode, funcNames)
	case parse.NodePipe:
		pipeNode := node.(*parse.PipeNode)
		return matchPipeNode(pipeNode, funcNames)
	case parse.NodeCommand:
		cmdNode := node.(*parse.CommandNode)
		return matchCmdNode(cmdNode, funcNames)
	case parse.NodeList:
		listNode := node.(*parse.ListNode)
		return matchListNode(listNode, funcNames)
	case parse.NodeIf:
		ifNode := node.(*parse.IfNode)
		return matchIfNode(ifNode, funcNames)
	case parse.NodeRange:
		rangeNode := node.(*parse.RangeNode)
		return matchRangeNode(rangeNode, funcNames)
	case parse.NodeWith:
		withNode := node.(*parse.WithNode)
		return matchWithNode(withNode, funcNames)
	case parse.NodeTemplate:
		templateNode := node.(*parse.TemplateNode)
		return matchTemplateNode(templateNode, funcNames)
	}
	return false
}

func getConfdKV(fileData interface{}, keys []string) (map[string]string, error) {
	vars := make(map[string]string)
	yamlMap := make(map[string]interface{})

	data, err := yaml2.Marshal(fileData)
	if err != nil {
		return nil, err
	}
	data = stripData(data)
	data, err = yaml2.YAMLToJSON(data)
	if err != nil {
		return nil, err
	}

	err = yaml2.Unmarshal(data, &yamlMap)
	if err != nil {
		return nil, err
	}

	err = nodeWalk(yamlMap, "/", vars)
	if err != nil {

	}
VarsLoop:
	for k, _ := range vars {
		for _, key := range keys {
			if strings.HasPrefix(k, key) {
				continue VarsLoop
			}
		}
		delete(vars, k)
	}
	klog.Infof(fmt.Sprintf("Key Map: %#v", vars))
	return vars, nil
}

func stripData(file []byte) []byte {
	stripped := []byte{}
	lines := bytes.Split(file, []byte("\n"))
	for i, line := range lines {
		if bytes.HasPrefix(bytes.TrimSpace(line), []byte("#")) || bytes.HasPrefix(bytes.TrimSpace(line), []byte("|")) || bytes.HasPrefix(bytes.TrimSpace(line), []byte("|-")) {
			continue
		}
		stripped = append(stripped, line...)
		if i < len(lines)-1 {
			stripped = append(stripped, '\n')
		}
	}
	return stripped
}
