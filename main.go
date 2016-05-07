package main

import (
	"archive/tar"
	"compress/gzip"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/github"
)

func main() {
	var (
		r = flag.String("r", "", "owner/repo")
		b = flag.String("b", "master", "branch")
	)
	flag.Parse()

	log.SetFlags(0)

	if *r == "" {
		log.Fatal("you must specify a owner/repo as -r argument")
	}

	repo := strings.Split(*r, "/")
	if len(repo) < 2 {
		log.Fatal("usage -r owner/repo")
	}

	url, err := getArchiveLink(repo[0], repo[1], *b)
	if err != nil {
		if _, ok := err.(*github.RateLimitError); ok {
			log.Fatal("Rate limit exceeded, please try again later.")
		}
		log.Fatalf("%s\n", err)
	}

	if url == nil {
		log.Fatalf("no URL found for %q\n", *r)
	}

	fmt.Printf("downloading from %s\n", url)
	filename := fmt.Sprintf("%s-%s.tar.gz", repo[0], repo[1], *b)

	go spinner(30 * time.Millisecond)
	if err := downloadFile(url.String(), filename); err != nil {
		log.Fatalf("%s\n", err)
	}

	if err := extract(filename); err != nil {
		log.Fatalf("%s\n", err)
	}

	if err := os.Remove(filename); err != nil {
		log.Fatalf("%s\n", err)
	}

	fmt.Println("")
}

func getArchiveLink(owner, repo, branch string) (*url.URL, error) {
	client := github.NewClient(nil)
	if branch == "" {
		branch = "master"
	}
	opt := &github.RepositoryContentGetOptions{Ref: branch}
	url, _, err := client.Repositories.GetArchiveLink(owner, repo, github.Tarball, opt)
	if err != nil {
		return nil, err
	}
	return url, nil
}

func downloadFile(url, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	resp, err := http.Get(url)
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}

	return nil
}

func extract(file string) error {

	gz, err := os.Open(file)
	if err != nil {
		return err
	}
	defer gz.Close()

	gr, err := gzip.NewReader(gz)
	if err != nil {
		return err
	}

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		name := header.Name
		switch header.Typeflag {
		case tar.TypeDir:
			err := os.MkdirAll(name, os.FileMode(header.Mode))
			if err != nil {
				return err
			}

		case tar.TypeReg:
			wr, err := os.Create(name)
			if err != nil {
				return err
			}
			io.Copy(wr, tr)

			err = os.Chmod(name, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			wr.Close()
		}
	}
	return nil
}

func spinner(delay time.Duration) {
	for {
		for _, r := range `-\|/` {
			fmt.Printf("\r%c", r)
			time.Sleep(delay)
		}
	}
}
