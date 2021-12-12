package main

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/creack/pty"
	"github.com/gliderlabs/ssh"
	"github.com/go-playground/validator/v10"
	"gopkg.in/yaml.v3"
)

var state struct {
	address  string
	username []byte
	password []byte
	token    string
	identity string
}

var validate = validator.New()

func setup() error {
	bytes, err := ioutil.ReadFile("config.yml")
	if err != nil {
		return err
	}
	var config struct {
		Address  string `validate:"required"`
		Username string `validate:"required"`
		Password string `validate:"required"`
		Token    string `validate:"required"`
		Identity string `validate:"required"`
	}
	err = yaml.Unmarshal(bytes, &config)
	if err != nil {
		return err
	}
	err = validate.Struct(config)
	if err != nil {
		return err
	}
	state.address = config.Address
	state.username = []byte(config.Username)
	state.password = []byte(config.Password)
	state.token = config.Token
	dir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	dir = path.Join(dir, "csjump")
	err = os.MkdirAll(dir, 0777)
	if err != nil {
		return err
	}
	state.identity = path.Join(dir, "identity")
	err = os.WriteFile(state.identity, []byte(config.Identity), 0600)
	if err != nil {
		return err
	}
	return nil
}

func handle(s ssh.Session) error {
	ptyReq, winCh, isPty := s.Pty()
	if !isPty {
		return errors.New("no terminal")
	}
	cmd := exec.CommandContext(s.Context(), "gh", "cs", "ssh", "--", "-i", state.identity)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("GITHUB_TOKEN=%v", state.token),
		fmt.Sprintf("TERM=%v", ptyReq.Term),
	)
	fp, err := pty.Start(cmd)
	if err != nil {
		return err
	}
	defer fp.Close()
	go io.Copy(fp, s)
	go io.Copy(s, fp)
	go func() {
		for win := range winCh {
			pty.Setsize(fp, &pty.Winsize{Rows: uint16(win.Height), Cols: uint16(win.Width), X: 0, Y: 0})
		}
	}()
	return cmd.Wait()
}

func main() {
	err := setup()
	if err != nil {
		log.Fatalln(err)
	}
	err = ssh.ListenAndServe(
		state.address,
		func(s ssh.Session) {
			err := handle(s)
			if err != nil {
				fmt.Fprintln(s.Stderr(), err)
			}
		},
		ssh.HostKeyFile(state.identity),
		ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
			return false
		}),
		ssh.PasswordAuth(func(ctx ssh.Context, password string) bool {
			var ok = true
			ok = ok && subtle.ConstantTimeCompare(state.username, []byte(ctx.User())) == 1
			ok = ok && subtle.ConstantTimeCompare(state.password, []byte(password)) == 1
			if !ok {
				time.Sleep(3 * time.Second)
			}
			return ok
		}),
	)
	log.Fatalln(err)
}
