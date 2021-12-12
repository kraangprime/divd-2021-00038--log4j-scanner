package app

import (

	// "github.com/dutchcoders/dirbuster/vendor.bak/gopkg.in/src-d/go-git.v4/utils/ioutil"

	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/gosuri/uilive"
	_ "github.com/op/go-logging"
)

type fuzzer struct {
	config

	writer *uilive.Writer
	output *writer

	dialer func(network, addr string) (net.Conn, error)

	cachePath string
	method    string

	targetHosts []string

	hosts []string

	proxyURL *url.URL

	template Template

	wordsCh chan string
}
type writer struct {
	f io.WriteCloser
	m sync.Mutex
}

func NewWriter(path string) (*writer, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	return &writer{
		f: f,
		m: sync.Mutex{},
	}, nil
}

func (w *writer) WriteLine(format string, args ...interface{}) {
	w.m.Lock()
	defer w.m.Unlock()

	fmt.Fprintln(w.f, fmt.Sprintf(format, args...))
}

func New(options ...OptionFn) (*fuzzer, error) {
	words := make(chan string)

	b := &fuzzer{
		wordsCh: words,
		config: config{
			suffixes: []string{},
		},
	}

	for _, optionFunc := range options {
		if err := optionFunc(b); err != nil {
			return nil, err
		}
	}

	if len(b.targetHosts) == 0 {
		return nil, fmt.Errorf("No target hosts set, nothing to do")
	}

	go func() {
		for _, target := range b.targetHosts {
			err := filepath.Walk(target, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				words <- path
				return nil
			})

			if err != nil {
				fmt.Printf("error walking the path %q: %v\n", target, err)
				return
			}
		}

		close(words)
	}()

	return b, nil
}

type unbufferedReaderAt struct {
	R io.Reader
	N int64
}

func NewUnbufferedReaderAt(r io.Reader) io.ReaderAt {
	return &unbufferedReaderAt{R: r}
}

func (u *unbufferedReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off < u.N {
		return 0, errors.New("invalid offset")
	}
	diff := off - u.N
	written, err := io.CopyN(ioutil.Discard, u.R, diff)
	u.N += written
	if err != nil {
		return 0, err
	}

	n, err = u.R.Read(p)
	u.N += int64(n)
	return
}

func (b *fuzzer) RecursiveFind(w []string, r *zip.Reader) error {
	// should check for hashes if vulnerable or not
	for _, f := range r.File {
		if f.Name == "org/apache/logging/log4j/core/lookup/JndiLookup.class" {
			fmt.Fprintln(b.writer.Bypass(), color.RedString("[!][%s] found JndiLookup: [%s]\u001b[0K", strings.Join(w, " -> "), "FOUND"))
		}

		func() error {
			rc, err := f.Open()
			if err != nil {
				return err
			}

			defer rc.Close()

			buff := bytes.NewBuffer([]byte{})

			size, err := io.Copy(buff, rc)
			if err != nil {
				return err
			}

			//ura := NewUnbufferedReaderAt(rc)

			br := bytes.NewReader(buff.Bytes())

			r2, err := zip.NewReader(br, size)
			if err != nil {
				return err
			}

			return b.RecursiveFind(append(w, f.Name), r2)
		}()
	}

	return nil
}

func (b *fuzzer) Run() error {
	b.writer = uilive.New()

	b.writer.Start()
	defer b.writer.Stop() // flush and stop rendering

	for w := range b.wordsCh {
		func() error {
			r, err := os.OpenFile(w, os.O_RDONLY, 0)
			if err != nil {
				return err
			}

			fi, err := r.Stat()
			if err != nil {
				return err
			}

			defer r.Close()

			r2, err := zip.NewReader(r, fi.Size())
			if err != nil {
				return err
			}

			return b.RecursiveFind([]string{w}, r2)
		}()
	}

	return nil
}
