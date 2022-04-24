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
	"sync"

	kustomizev1beta1 "github.com/fluxcd/kustomize-controller/api/v1beta1"
	kustomizev1beta2 "github.com/fluxcd/kustomize-controller/api/v1beta2"
	flag "github.com/spf13/pflag"
	_ "embed"
)

//go:embed templates/github-action
var githubActionRawTemplate string

type GitopsNode struct {
	Path string
	Children []*GitopsNode
	Bytes []byte
}

type Entrypoint struct {
	Path string
	Ignores []string
}

func newEntrypoint(v *viper.Viper) *Entrypoint {
	return &Entrypoint{
		Path: v.GetString("path"),
		Ignores: v.GetStringSlice("ignores"),
	}
}

type DiffMap struct {
	Path string
	Diff map[string]string
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

		allDiffsMap := map[string]map[string]string{}
		diffMaps := make(chan *DiffMap)
		var wg sync.WaitGroup
		for _, entryPoint := range entryPoints {
			currentEntryPoint := entryPoint
			firstEntrypointRoot := "old/"
			secondEntrypointRoot := "new/"

			firstEntrypoint := entryPoint.Path	
			secondEntrypoint := entryPoint.Path

			wg.Add(1)
			go func() {
				defer wg.Done()
				firstEntrypointOutput, err := buildKustomizeWithPath(firstEntrypointRoot, firstEntrypoint)
				if err != nil {
					panic(err)
				}
				
				firstExpanded, err := expandEntrypoint(firstEntrypointOutput.Bytes(), firstEntrypointRoot, currentEntryPoint.Ignores)
				if err != nil {
					panic(err)
				}

				secondEntrypointOutput, err := buildKustomizeWithPath(secondEntrypointRoot, secondEntrypoint)
				if err != nil {
					panic(err)
				}
				secondExpanded, err := expandEntrypoint(secondEntrypointOutput.Bytes(), secondEntrypointRoot, currentEntryPoint.Ignores)
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
			}()
		}
		for i := 0; i < len(entryPoints); i++ {
			currentDiffMap := <-diffMaps
			if len(currentDiffMap.Diff) > 0 {
				allDiffsMap[currentDiffMap.Path] = currentDiffMap.Diff
			}
		}
		wg.Wait()

		githubActionTemplate := template.Must(template.New("github-action").Funcs(sprig.TxtFuncMap()).Parse(githubActionRawTemplate))
		err := githubActionTemplate.Execute(os.Stdout, allDiffsMap)
		if err != nil {
			panic(err)
		}
	},
}

var ignoredPathFlag []string
func init() {
	var cfgFile string
	flag.StringArrayVar(&ignoredPathFlag, "ignore", []string{}, "Relative directory paths that should not be expanded.")
	flag.StringVar(&cfgFile, "config", "config.yml", "The path to the config file to be used.")

	viper.SetConfigType("yaml")
	viper.SetConfigFile(cfgFile)
	err := viper.ReadInConfig() 
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}

	kustomizev1beta1.AddToScheme(scheme.Scheme)
	kustomizev1beta2.AddToScheme(scheme.Scheme)
	rootCmd.AddCommand(diffCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// diffCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// diffCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func dyffExpandedTree(firstTree *GitopsNode, secondTree *GitopsNode) (map[string]string, error) {
	diffMap := map[string]string{}
	var matchedPaths []string
	for _, firstChild := range firstTree.Children {
		for count, secondChild := range secondTree.Children {
			if firstChild.Path == secondChild.Path {
				matchedPaths = append(matchedPaths, firstChild.Path)
				dyffOutput, err := dyffKustomize(firstChild.Bytes, secondChild.Bytes)
				if err != nil {
					panic(err)
				}
				if len(dyffOutput.Bytes()) > 1 {
					diffMap[firstChild.Path] = string(dyffOutput.Bytes())
				}
				break
			}
			// If we didn't find the firstChild path compare against an empty file, catches deletions
			if count == len(secondTree.Children) + 1 {
				matchedPaths = append(matchedPaths, firstChild.Path)
				dyffOutput, err := dyffKustomize(firstChild.Bytes, []byte{})
				if err != nil {
					panic(err)
				}
				if len(dyffOutput.Bytes()) > 1 {
					diffMap[firstChild.Path] = string(dyffOutput.Bytes())
				}
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
		dyffOutput, err := dyffKustomize([]byte{}, secondChild.Bytes)
		if err != nil {
			panic(err)
		}
		diffMap[secondChild.Path] = string(dyffOutput.Bytes())
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

func expandEntrypoint(entrypoint []byte, kustomizePathRoot string, ignoredPaths []string) (*GitopsNode, error) {
	currentNode := &GitopsNode{
		Children: []*GitopsNode{},
		Bytes: entrypoint,
	}
	allDocuments, err := splitDocuments(entrypoint)
	if err != nil {
		return nil, err
	}
	for _, document := range allDocuments {
		var nextPath *string
		// Convert manifest to runtime.Object
		m, _, err := scheme.Codecs.UniversalDeserializer().Decode(document, nil, nil)
		if err != nil {
			continue
		}
		if m.GetObjectKind().GroupVersionKind().Kind == "Kustomization" {
			if m.GetObjectKind().GroupVersionKind().Version == "v1beta1" {
				fluxKustomization, ok := m.(*kustomizev1beta1.Kustomization)
				if !ok {
					panic(errors.New("Kustomize v1beta1 detected but cannot be parsed"))
				}
				nextPath = &fluxKustomization.Spec.Path
			} else if m.GetObjectKind().GroupVersionKind().Version == "v1beta2" {
				fluxKustomization, ok := m.(*kustomizev1beta2.Kustomization)
				if !ok {
					panic(errors.New("Kustomize v1beta2 detected but cannot be parsed"))
				}
				nextPath = &fluxKustomization.Spec.Path
			} else {
				panic(errors.New("Kustomize detected but cannot be parsed"))
			}
		}

		// Skip to next document if no path is set
		if nextPath == nil {
			continue
		}
		var pathIsIgnored bool
		for _, ignoredPath := range ignoredPaths {
			if *nextPath == ignoredPath {
				pathIsIgnored = true
			}
		}

		if !pathIsIgnored {
			entrypointOutput, err := buildKustomizeWithPath(kustomizePathRoot, *nextPath)
			if err != nil {
				fmt.Println("failed to build")
				return nil, err
			}
			entrypointExpanded, err := expandEntrypoint(entrypointOutput.Bytes(), kustomizePathRoot, ignoredPaths)
			if err != nil {
				fmt.Println("Failed to expand")
				return nil, err
			}
			entrypointExpanded.Path = *nextPath
			currentNode.Children = append(currentNode.Children, entrypointExpanded)
		}
	}
	
	return currentNode, nil
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
