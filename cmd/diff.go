/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"path/filepath"
	"io"
	"os"
	"fmt"
	"errors"
	"github.com/spf13/cobra"
	"bytes"
	"os/exec"
	"io/ioutil"
	"gopkg.in/yaml.v3"
	"k8s.io/client-go/kubernetes/scheme"
	"golang.org/x/exp/maps"
	"strings"
	"text/template"
	"github.com/spf13/viper"
	"github.com/Masterminds/sprig"

	kustomizev1beta1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	kustomizev1beta2 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	applicationv1alpha1 "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	flag "github.com/spf13/pflag"
	_ "embed"
)

//go:embed templates/github-action
var githubActionRawTemplate string

var cfgFile string

type GitopsNode struct {
	Path string
	Children []*GitopsNode
	Bytes []byte
}

type Entrypoint struct {
	Path string
}

func newEntrypoint(v *viper.Viper) *Entrypoint {
	return &Entrypoint{
		Path: v.GetString("path"),
	}
}

type DiffMap struct {
	Path string
	Diff map[string]string
}

type Diff struct {
	Path string
	DiffOutput []byte
}

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "aaaaaa",
	Long: `aaaaaaaaaaa`,
	Args: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	Run: func(cmd *cobra.Command, args []string) {
		var entryPoints []*Entrypoint
		// lol
		for count, _ := range viper.Get("entryPoints").([]interface{}) {
			viperSubAddress := fmt.Sprintf("entryPoints.%v", count)
			entryPoints = append(entryPoints, newEntrypoint(viper.Sub(viperSubAddress)))
		}

		var goCount int
		allDiffsMap := map[string]map[string]string{}
		diffMaps := make(chan *DiffMap)
		for _, entryPoint := range entryPoints {
			currentEntryPoint := entryPoint
			firstEntrypointRoot := "old/"
			secondEntrypointRoot := "new/"

			firstEntrypoint := entryPoint.Path	
			secondEntrypoint := entryPoint.Path

			goCount += 1
			go func(firstEntrypointRoot string, secondEntrypointRoot string, firstEntrypoint string, secondEntrypoint string) {
				firstExpanded, _, err := expandEntrypoint(firstEntrypointRoot, firstEntrypoint, []string{})
				if err != nil {
					panic(err)
				}

				secondExpanded, _, err := expandEntrypoint(secondEntrypointRoot, secondEntrypoint, []string{})
				if err != nil {
					panic(err)
				}

				diffMap := map[string]string{}
				// Dyff the first tree
				dyffOutput, err := dyffKustomize(firstExpanded.Bytes, secondExpanded.Bytes)
				if err != nil {
					panic(err)
				}

				diffMap[firstEntrypoint] = string(dyffOutput.Bytes())
				childDiffMap, err := dyffExpandedTree(firstExpanded, secondExpanded)
				if err != nil {
					panic(err)
				}
				if len(dyffOutput.Bytes()) > 1 {
					maps.Copy(childDiffMap, diffMap)
				}
				diffMaps <- &DiffMap{
					Path: currentEntryPoint.Path,
					Diff: childDiffMap,
				}
			}(firstEntrypointRoot, secondEntrypointRoot, firstEntrypoint, secondEntrypoint)
		}

		for i := 0; i < goCount; i++ {
			currentDiffMap := <-diffMaps
			if len(currentDiffMap.Diff) > 0 {
				allDiffsMap[currentDiffMap.Path] = currentDiffMap.Diff
			}
		}

		githubActionTemplate := template.Must(template.New("github-action").Funcs(sprig.TxtFuncMap()).Parse(githubActionRawTemplate))
		err := githubActionTemplate.Execute(os.Stdout, allDiffsMap)
		if err != nil {
			panic(err)
		}
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	flag.StringVar(&cfgFile, "config", "config.yml", "The path to the config file to be used.")

	kustomizev1beta1.AddToScheme(scheme.Scheme)
	kustomizev1beta2.AddToScheme(scheme.Scheme)
	applicationv1alpha1.AddToScheme(scheme.Scheme)

	rootCmd.AddCommand(diffCmd)
}

func initConfig() {
	viper.SetConfigType("yaml")
	viper.SetConfigFile(cfgFile)
	err := viper.ReadInConfig() 
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}
}


func dyffExpandedTree(firstTree *GitopsNode, secondTree *GitopsNode) (map[string]string, error) {
	diffMap := map[string]string{}
	var matchedPaths []string

	var goCount int
	diffs := make(chan *Diff)
	for _, firstChild := range firstTree.Children {
		for count, secondChild := range secondTree.Children {
			if firstChild.Path == secondChild.Path {
				matchedPaths = append(matchedPaths, firstChild.Path)
				goCount += 1
				go func(firstChild *GitopsNode, secondChild *GitopsNode) {
					fmt.Println(firstChild.Path, secondChild.Path)
					dyffOutput, err := dyffKustomize(firstChild.Bytes, secondChild.Bytes)
					if err != nil {
						panic(err)
					}
					diffs <- &Diff{
						Path: firstChild.Path,
						DiffOutput: dyffOutput.Bytes(),
					}
				}(firstChild, secondChild)
				break
			}
			// If we didn't find the firstChild path compare against an empty file, catches deletions
			if count == len(secondTree.Children) + 1 {
				matchedPaths = append(matchedPaths, firstChild.Path)
				goCount += 1
				go func(secondChild *GitopsNode) {
					dyffOutput, err := dyffKustomize(firstChild.Bytes, []byte{})
					if err != nil {
						panic(err)
					}
					diffs <- &Diff{
						Path: firstChild.Path,
						DiffOutput: dyffOutput.Bytes(),
					}
				}(secondChild)
			}
		}
	}

	// Go in the other direction to catch additions in the new tree not present in the old
	var skip bool
	for _, secondChild := range secondTree.Children {
		for _, matchedPath := range matchedPaths {
			if secondChild.Path == matchedPath {
				skip = true
				break
			}
		}
		if skip {
			skip = false
			continue
		}
		goCount += 1
		go func(secondChild *GitopsNode) {
			dyffOutput, err := dyffKustomize([]byte{}, secondChild.Bytes)
			if err != nil {
				panic(err)
			}
			diffs <- &Diff{
				Path: secondChild.Path,
				DiffOutput: dyffOutput.Bytes(),
			}
		}(secondChild)
	}
	for i := 0; i < goCount; i++ {
		diff := <-diffs
		if len(diff.DiffOutput) > 1 {
			diffMap[diff.Path] = string(diff.DiffOutput)
		}
	}
	return diffMap, nil
}

func buildKustomizeWithPath(kustomizePathRoot string, kustomizePathRelative string) (*bytes.Buffer, error) {
	kustomizePath := filepath.Join(kustomizePathRoot, kustomizePathRelative)
	// Catch stdout in buffer
	kustomizeStdout := &bytes.Buffer{}

	kustomizeArgs := []string{"build", kustomizePath}
	kustomizeCommand := exec.Command("kustomize", kustomizeArgs...)
	kustomizeCommand.Stdout = kustomizeStdout
	kustomizeCommand.Stderr = os.Stderr
	err := kustomizeCommand.Run()
	if err != nil {
		return nil, err
	}
	return kustomizeStdout, nil
}

func dyffKustomize(oldBytes []byte, newBytes []byte) (*bytes.Buffer, error) {
	oldContentTempFile, err := writeBytesToTempfile(oldBytes)
	defer os.Remove(oldContentTempFile.Name())
	if err != nil {
		return nil, err
	}
	newContentTempFile, err := writeBytesToTempfile(newBytes)
	defer os.Remove(newContentTempFile.Name())
	if err != nil {
		return nil, err
	}
	// Catch stdout in buffer
	stdOut := &bytes.Buffer{}

	dyffStderr := &bytes.Buffer{}

	dyffArgs := []string{"between", "-c", "off", "--ignore-order-changes", "--omit-header", oldContentTempFile.Name(), newContentTempFile.Name()}
	dyffCommand := exec.Command("dyff", dyffArgs...)
	dyffCommand.Stdout = stdOut
	dyffCommand.Stderr = dyffStderr
	err = dyffCommand.Run()
	if err != nil {
		// Use diff when dyff fails us
		if strings.Contains(string(dyffStderr.Bytes()), "comparing YAMLs with a different number of documents") {
			diffStderr := &bytes.Buffer{}

			diffArgs := []string{oldContentTempFile.Name(), newContentTempFile.Name()}
			diffCommand := exec.Command("diff", diffArgs...)
			diffCommand.Stdout = stdOut
			diffCommand.Stderr = diffStderr
			err = diffCommand.Run()
			if err != nil {
				// diff has an exit code of 1 when changes are found
				if exitError, ok := err.(*exec.ExitError); ok {
					if exitError.ExitCode() == 1 {
						return stdOut, nil
					}
				} else {
					fmt.Println(diffStderr.Bytes())
					return nil, err
				}
			}
		} else {
			fmt.Println(dyffStderr.Bytes())
			return nil, err
		}
	}
	return stdOut, nil
}


func writeBytesToTempfile(inBytes []byte) (*os.File, error) {
	tempFile, err := ioutil.TempFile("", "DiphTempFile.*.yml")
	if err != nil {
		return nil, err
	}
	defer tempFile.Close()

	bytes.NewReader(inBytes).WriteTo(tempFile)
	tempFile.Sync()
	return tempFile, nil
}

func expandEntrypoint(kustomizePathRoot string, kustomizePath string, processedPaths []string) (*GitopsNode, []string, error) {
	entrypointOutput, err := buildKustomizeWithPath(kustomizePathRoot, kustomizePath)
	if err != nil {
		fmt.Println("failed to build")
		return nil, []string{}, err
	}

	currentNode := &GitopsNode{
		Children: []*GitopsNode{},
		Bytes: entrypointOutput.Bytes(),
	}

	allDocuments, err := splitDocuments(entrypointOutput.Bytes())
	if err != nil {
		return nil, []string{}, err
	}
	for _, document := range allDocuments {
		var nextPath *string
		// Convert manifest to runtime.Object
		m, _, err := scheme.Codecs.UniversalDeserializer().Decode(document, nil, nil)
		if err != nil {
			continue
		}

		objectGVK := m.GetObjectKind().GroupVersionKind()
		if objectGVK.Kind == "Kustomization" {
			if objectGVK.Version == "v1beta1" {
				fluxKustomization, ok := m.(*kustomizev1beta1.Kustomization)
				if !ok {
					panic(errors.New("Flux2 Kustomization v1beta1 detected but cannot be parsed"))
				}
				nextPath = &fluxKustomization.Spec.Path
			} else if objectGVK.Version == "v1beta2" {
				fluxKustomization, ok := m.(*kustomizev1beta2.Kustomization)
				if !ok {
					panic(errors.New("Flux2 Kustomization v1beta2 detected but cannot be parsed"))
				}
				nextPath = &fluxKustomization.Spec.Path
			} else {
				panic(errors.New("Kustomization detected but cannot be parsed"))
			}
		} else if objectGVK.Kind == "Application" {
			if objectGVK.Version == "v1alpha1" {
				argocdApplication, ok := m.(*applicationv1alpha1.Application)
				if !ok {
					panic(errors.New("ArgoCD Application v1alpha1 detected but cannot be parsed"))
				}
				if argocdApplication.Spec.Source.Path != "" {
					nextPath = &argocdApplication.Spec.Source.Path
				}
			} else {
				panic(errors.New("ArgoCD Application detected but cannot be parsed"))
			}
		}

		// Skip to next document if no path is set
		if nextPath == nil {
			continue
		}

		// Prevent infinite recursion
		var pathIsProcessed bool
		for _, processedPath := range processedPaths {
			if *nextPath == processedPath {
				pathIsProcessed = true
			}
		}

		if !pathIsProcessed {
			// Add self to block list for next run
			processedPaths = append(processedPaths, *nextPath)
			entrypointExpanded, newProcessedPaths, err := expandEntrypoint(kustomizePathRoot, *nextPath, processedPaths)
			if err != nil {
				fmt.Println("Failed to expand")
				return nil, []string{}, err
			}
			entrypointExpanded.Path = *nextPath
			currentNode.Children = append(currentNode.Children, entrypointExpanded)

			// Add all processed paths to block list
			processedPaths = append(processedPaths, newProcessedPaths...)
		}
	}
	
	return currentNode, []string{}, nil
}

func splitDocuments(documents []byte) ([][]byte, error) {
	reader := bytes.NewReader(documents)
    dec := yaml.NewDecoder(reader)

	var splitDocuments [][]byte
    for {
        var node yaml.Node
        err := dec.Decode(&node)
        if errors.Is(err, io.EOF) {
            break
        }
        if err != nil {
            return splitDocuments, err
        }

        content, err := yaml.Marshal(&node)
        if err != nil {
            return splitDocuments, err
        }
		splitDocuments = append(splitDocuments, content)
    }
	return splitDocuments, nil
}
