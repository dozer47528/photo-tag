package main

import (
	"fmt"
	"os/user"
	"os"
	"encoding/json"
	"io/ioutil"
	"github.com/TencentYouTu/go_sdk"
	"os/exec"
	"bytes"
	"flag"
	"strings"
	"path"
	"image"
	"github.com/nfnt/resize"
	"image/jpeg"
	"bufio"
	"regexp"
)

var pathArgs = flag.String("p", "./", "Photo file paht")
var deleteOldTagArgs = flag.Bool("d", false, "Delete current tags")

func main() {
	flag.Parse()

	yt := initYoutu()

	imageFiles := fetchImageFiles()

	for _, f := range imageFiles {
		tagMap := loadTags(f)

		dataOrigin, _ := os.Open(f)
		imgOrigin, _, _ := image.Decode(dataOrigin)
		dataOrigin.Close()

		imgDst := resize.Resize(800, 0, imgOrigin, resize.Lanczos3)

		var buff bytes.Buffer
		_ = jpeg.Encode(bufio.NewWriter(&buff), imgDst, nil)
		tagResponse, err := yt.ImageTag(buff.Bytes(), 0, "")
		if err != nil {
			fmt.Fprintf(os.Stderr, "ImageTag() failed: %s", err)
			os.Exit(1)
		}

		cmds := make([]string, 0)

		if *deleteOldTagArgs {
			cmds = append(cmds, "-M", "del Iptc.Application2.Keywords")
		}

		tagNames := make([]string, 0)
		for _, tag := range tagResponse.Tags {
			if !*deleteOldTagArgs && tagMap[tag.TagName] {
				continue
			}

			cmd := "add Iptc.Application2.Keywords String " + tag.TagName
			cmds = append(cmds, "-M", cmd)
			tagNames = append(tagNames, tag.TagName)
		}

		if len(cmds) == 0 {
			fmt.Printf("%s: Skip\n", f)
			continue
		}

		cmds = append(cmds, f)
		command := exec.Command("exiv2", cmds...)

		if err := command.Run(); err != nil {
			fmt.Println(err)
		}

		fmt.Printf("%s: %s\n", f, strings.Join(tagNames, ","))
	}
}

func fetchImageFiles() []string {
	files, _ := ioutil.ReadDir(*pathArgs)
	images := make([]string, 0)
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		lowerName := strings.ToLower(f.Name())
		if strings.HasSuffix(lowerName, ".jpg") || strings.HasSuffix(lowerName, ".jpeg") {
			images = append(images, path.Join(*pathArgs, f.Name()))
		}
	}
	return images
}

func loadTags(filename string) map[string]bool {
	tagMap := make(map[string]bool)

	cmd := exec.Command("exiv2", "-PI", filename)
	out := bytes.NewBufferString("")
	cmd.Stdout = out

	if err := cmd.Run(); err != nil {
		return tagMap
	}

	output := out.String()
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		re, _ := regexp.Compile("^Iptc\\.Application2\\.Keywords\\s+String\\s+\\d+\\s+(.+)$")
		result := re.FindStringSubmatch(line)
		if len(result) >= 2 {
			tagMap[result[1]] = true
		}
	}

	return tagMap
}

func initYoutu() *youtu.Youtu {
	usr, err := user.Current()
	if err != nil {
		fmt.Printf("Can't get current user: %v\n", err)
		os.Exit(1)
	}

	filePath := usr.HomeDir + "/.youtu.json"
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Open youtu config file error: %v\n", err)
		os.Exit(1)
	}

	var settings struct {
		AppId     int
		SecretId  string
		SecretKey string
	}

	err = json.Unmarshal(file, &settings)
	if err != nil {
		fmt.Println("Youtu config file read failed:", err)
		fmt.Println("Config should be json: \n{\n" + "    \"appId\": 10000000,\n" + "    \"secretId\": \"\",\n" + "    \"secretKey\": \"\"\n}")
		os.Exit(1)
	}

	//Get the following details
	appID := uint32(settings.AppId)
	secretID := settings.SecretId
	secretKey := settings.SecretKey
	userID := "Dozer"

	as, err := youtu.NewAppSign(appID, secretID, secretKey, userID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "NewAppSign() failed: %s\n", err)
		os.Exit(1)
	}
	return youtu.Init(as, youtu.DefaultHost)
}
