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
)

var pathArgs = flag.String("p", "./", "Photo file paht.")

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
		fmt.Println("Config should be json: \n{\n"+"    \"appId\": 10000000,\n"+"    \"secretId\": \"\",\n"+"    \"secretKey\": \"\"\n}")
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

func main() {
	flag.Parse()

	yt := initYoutu()

	imageFiles := fetchImageFiles()

	for _, f := range imageFiles {
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
		cmds = append(cmds, "-M", "del Iptc.Application2.Keywords")
		tagNames := make([]string, 0)
		for _, tag := range tagResponse.Tags {
			cmd := "add Iptc.Application2.Keywords String " + tag.TagName
			cmds = append(cmds, "-M", cmd)
			tagNames = append(tagNames, tag.TagName)
		}
		cmds = append(cmds, f)
		command := exec.Command("exiv2", cmds...)

		var out bytes.Buffer
		command.Stdout = &out
		if err := command.Run(); err != nil {
			fmt.Println(err)
		}

		fmt.Printf("OK: %s %s\n", f, strings.Join(tagNames, ","))
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
