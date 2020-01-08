package main

import (
	"bufio"
	"fmt"
	"github.com/jeremywohl/flatten"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const flagPattern = "pattern"
const flagSource = "source"
const flagTarget = "target"

var (
	newLines = regexp.MustCompile(`\r|\n`)
	braces   = regexp.MustCompile(`\{|\}`)
)

func main() {

	name := "Generator"
	runner := cli.NewApp()
	runner.Usage = name
	runner.Version = "1.0"

	runner.Commands = []*cli.Command{
		{
			Name:  "json2csv",
			Usage: "Flatten JSON structure to CSV",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  fmt.Sprintf("%v, %v", flagPattern, "p"),
					Usage: "RegExp pattern with named groups, e.g. (?s)(?P<label>.+?)(?P<json>\\{.+?\\})\"",
				},
				&cli.StringFlag{
					Name:  fmt.Sprintf("%v, %v", flagSource, "s"),
					Usage: "Source file or folder",
				},
				&cli.StringFlag{
					Name:  fmt.Sprintf("%v, %v", flagTarget, "t"),
					Usage: "Target file or folder",
				},
			},
			Action: func(c *cli.Context) (err error) {
				l(c).Info("convert json to flatten csv")

				json2csv(c.String(flagPattern), c.String(flagSource), c.String(flagTarget))

				return
			},
		},
	}

	if err := runner.Run(os.Args); err != nil {
		log.Infof("run failed, %v, %v", os.Args, err)
	}
	log.Infof("done %v", os.Args)
}

func l(c *cli.Context) *log.Entry {
	return log.WithFields(log.Fields{
		flagPattern: c.String(flagPattern),
		flagSource:  c.String(flagSource),
		flagTarget:  c.String(flagTarget),
	})
}

func json2csv(pattern string, sourceFile string, targetFile string) {
	//pattern = `(?s)(?P<label>.+?)(?P<json>\{.+?\})\"`
	groupPattern := regexp.MustCompile(pattern)

	sourceFileInfo, err := os.Stat(sourceFile)
	if err != nil {
		log.Fatalf("can't touch %v, %v", sourceFile, err)
	}

	targetFileInfo, err := os.Stat(targetFile)
	if err != nil {
		log.Fatalf("can't touch %v, %v", targetFile, err)
	}

	if sourceFileInfo.IsDir() && targetFileInfo.IsDir() {
		err = filepath.Walk(sourceFile, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				json2Csv(groupPattern, path, filepath.Join(targetFile, info.Name()))
			}
			return nil
		})
		if err != nil {
			log.Fatalf("can't walk in %v, %v", sourceFile, err)
		}
	} else if !sourceFileInfo.IsDir() && !targetFileInfo.IsDir() {
		json2Csv(groupPattern, sourceFile, targetFile)
	} else {
		log.Fatalf("source and target files must be both folder or both files, %v, %v", sourceFile, targetFile)
	}
}

func json2Csv(groupPattern *regexp.Regexp, sourceFile string, targetFile string) {
	source, err := ioutil.ReadFile(sourceFile)
	if err != nil {
		log.Fatalf("can't open %v, %v", sourceFile, err)
	}
	sourceSrc := string(source)
	sourceSrc = strings.Replace(sourceSrc, "\"\"", "\"", -1)
	n1 := groupPattern.SubexpNames()
	r1 := groupPattern.FindAllStringSubmatch(sourceSrc, -1)

	var realGroups []string
	for _, groupName := range n1 {
		if groupName != "" {
			realGroups = append(realGroups, groupName)
		}
	}

	if r1 != nil {
		target, err := os.Create(targetFile)
		if err != nil {
			log.Fatalf("can't create target file %v, %v", targetFile, err)
		}

		targetWriter := bufio.NewWriter(target)
		for _, item := range r1 {
			for i, n := range item {
				group := n1[i]
				if group != "" {
					data := removeNewLines(n)

					targetWriter.WriteString(group)
					targetWriter.WriteString(";")

					flat, err := flatten.FlattenString(data, "", flatten.DotStyle)
					if err == nil {
						flat = strings.Replace(flat, "\":", "\";", -1)
						flat = braces.ReplaceAllString(flat, "")
						flat = strings.Replace(flat, ",", ";", -1)
						targetWriter.WriteString(flat)
						//targetWriter.WriteString(simicolonToUndescore.ReplaceAllString(csvStr,"_"))
						//targetWriter.WriteString(flat)
					} else {
						log.Debug("can't flatten %v", group, err)
						data = newLines.ReplaceAllString(data, "")
						targetWriter.WriteString(data)
					}
					targetWriter.WriteString(";")
				}
			}
			targetWriter.WriteString("\n")
		}
		err = targetWriter.Flush()
		if err != nil {
			log.Fatalf("can't flush target file %v, %v", targetFile, err)
		}
		log.Infof("file written %v", targetFile)
	} else {
		log.Warnf("no matches in %v", sourceFile)
	}
}

func removeNewLines(text string) string {
	return newLines.ReplaceAllString(text, " ")
}
