package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"

	"github.com/docker/docker/client"
)

func main() {
	code := `
package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	// Seeding with the same value results in the same random sequence each run.
	// For different numbers, seed with a different value, such as
	// time.Now().UnixNano(), which yields a constantly-changing number.
	rand.Seed(time.Now().UnixNano())
	answers := []string{
		"It is certain",
		"It is decidedly so",
		"Without a doubt",
		"Yes definitely",
		"You may rely on it",
		"As I see it yes",
		"Most likely",
		"Outlook good",
		"Yes",
		"Signs point to yes",
		"Reply hazy try again",
		"Ask again later",
		"Better not tell you now",
		"Cannot predict now",
		"Concentrate and ask again",
		"Don't count on it",
		"My reply is no",
		"My sources say no",
		"Outlook not so good",
		"Very doubtful",
	}
	fmt.Println("Magic 8-Ball says:", answers[rand.Intn(len(answers))])
}

	`
	New(code)
}

func New(code string) {
	err := writeCodeToFile(code)

	if err != nil {
		panic(err)
	}

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	buildImage(cli)
	x := buildContainer(cli)

	y := getLogs(x, cli)
	fmt.Println(y)

	cleanup(cli, "test")

}

func buildImage(cli *client.Client) {

	config := types.ImageBuildOptions{Tags: []string{"gotest"}}
	createBuildContext()
	buildContext, err := os.Open("./tmp/tar/go.tar")
	if err != nil {
		panic(err)
	}
	defer buildContext.Close()

	br, err := cli.ImageBuild(context.Background(), buildContext, config)
	if err != nil {
		panic(err)
	}
	defer br.Body.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(br.Body)
	s := buf.String()
	fmt.Println(s)
	fmt.Println("end of buildimage")

}

func createBuildContext() {
	file, err := os.Create("/home/jake/apps/gopherpun/runner/tmp/tar/go.tar")
	if err != nil {
		panic(err)
	}

	gzipWriter := gzip.NewWriter(file)
	defer gzipWriter.Close()

	tw := tar.NewWriter(gzipWriter)
	defer tw.Close()

	files := map[string][]byte{"Dockerfile": nil, "main.go": nil}
	for k, v := range files {
		a, err := ioutil.ReadFile("/home/jake/apps/gopherpun/runner/tmp/go/" + k)
		v = a
		if err != nil {
			panic(err)
		}
		hdr := &tar.Header{Name: k, Mode: 0600, Size: int64(len(v))}

		if err := tw.WriteHeader(hdr); err != nil {
			panic(err)
		}

		if _, err := tw.Write([]byte(v)); err != nil {
			panic(err)
		}

	}

}

func cleanup(cli *client.Client, id string) error {
	config := types.ContainerRemoveOptions{Force: true}
	ctx := context.Background()

	err := cli.ContainerRemove(ctx, id, config)

	if err != nil {
		return err
	}

	_, err = cli.ImageRemove(ctx, "gotest", types.ImageRemoveOptions{Force: true})
	if err != nil {
		return err
	}

	return nil
}

func writeCodeToFile(code string) error {
	path := "/home/jake/apps/gopherpun/runner/tmp/go/main.go"
	os.Remove(path)
	os.Create(path)
	f, err := os.OpenFile(path, os.O_RDWR, 0777)

	if err != nil {
		return err
	}

	if _, err := f.WriteString(code); err != nil {
		return err
	}

	return nil
}

func buildContainer(cli *client.Client) string {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config := container.Config{Image: "gotest"}
	hostconfig := container.HostConfig{}
	netconfig := network.NetworkingConfig{}

	c, err := cli.ContainerCreate(ctx, &config, &hostconfig, &netconfig, "test")
	if err != nil {
		panic(err)
	}
	if err := cli.ContainerStart(ctx, c.ID, types.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	return c.ID
}

func getLogs(id string, cli *client.Client) string {
	var b strings.Builder

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	reader, err := cli.ContainerLogs(ctx, id, types.ContainerLogsOptions{ShowStdout: true, ShowStderr: true})
	if err != nil {
		fmt.Println(err)
	}

	_, err = io.Copy(&b, reader)
	if err != nil && err != io.EOF {
		fmt.Println(err)
	}

	return b.String()
}
