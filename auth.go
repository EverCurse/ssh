package ssh

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
)

// An implementation of ssh.KeyboardInteractiveChallenge that simply sends
// back the password for all questions. The questions are logged.
func passwordKeyboardInteractive(password string) ssh.KeyboardInteractiveChallenge {
	return func(user, instruction string, questions []string, echos []bool) ([]string, error) {
		// log.Printf("Keyboard interactive challenge: ")
		// log.Printf("-- User: %s", user)
		// log.Printf("-- Instructions: %s", instruction)
		// for i, question := range questions {
		// 	log.Printf("-- Question %d: %s", i+1, question)
		// }

		// Just send the password back for all questions
		answers := make([]string, len(questions))
		for i := range answers {
			answers[i] = password
		}

		return answers, nil
	}
}

// WithKeyboardPassword Generate a password-auth'd ssh ClientConfig
func WithKeyboardPassword(password string) (ssh.AuthMethod, error) {
	return ssh.KeyboardInteractive(passwordKeyboardInteractive(password)), nil
}

// WithPassword Generate a password-auth'd ssh ClientConfig
func WithPassword(password string) (ssh.AuthMethod, error) {
	return ssh.Password(password), nil
}

// WithAgent use already authed user
func WithAgent() (ssh.AuthMethod, error) {
	sock := os.Getenv("SSH_AUTH_SOCK")
	if sock != "" {
		// fmt.Println(errors.New("Agent Disabled"))
		return nil, errors.New("Agent Disabled")
	}
	socks, err := net.Dial("unix", sock)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	// 1. 返回Signers函数的结果
	agent := agent.NewClient(socks)
	signers, err := agent.Signers()
	return ssh.PublicKeys(signers...), nil
	// 2. 返回Signers函数
	// getSigners := agent.NewClient(socks).Signers
	// return ssh.PublicKeysCallback(getSigners), nil

	// 3.简写方式
	// if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
	// 	return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	// }
	// return nil
}

// WithPrivateKeys 设置多个 ~/.ssh/id_rsa
func WithPrivateKeys(keyFiles []string, password string) (ssh.AuthMethod, error) {
	var signers []ssh.Signer

	for _, key := range keyFiles {

		buffer, err := ioutil.ReadFile(key)
		if err != nil {
			println(err.Error())
			// return
		}
		signer, err := ssh.ParsePrivateKeyWithPassphrase([]byte(buffer), []byte(password))
		if err != nil {
			println(err.Error())
		} else {
			signers = append(signers, signer)
		}
	}
	if signers == nil {
		return nil, errors.New("WithPrivateKeys: no keyfiles input")
	}
	return ssh.PublicKeys(signers...), nil
}

// WithPrivateKey 自动监测是否带有密码
func WithPrivateKey(keyfile string, password string) (ssh.AuthMethod, error) {
	pemBytes, err := ioutil.ReadFile(keyfile)
	if err != nil {
		println(err.Error())
		return nil, err
	}

	var signer ssh.Signer
	signer, err = ssh.ParsePrivateKey(pemBytes)
	if err != nil {
		if strings.Contains(err.Error(), "cannot decode encrypted private keys") {
			if signer, err = ssh.ParsePrivateKeyWithPassphrase(pemBytes, []byte(password)); err == nil {
				return ssh.PublicKeys(signer), nil
			}
		}
	}
	return nil, err
}

// WithPrivateKeyString 直接通过字符串
func WithPrivateKeyString(key string, password string) (ssh.AuthMethod, error) {
	var signer ssh.Signer
	var err error
	if password == "" {
		signer, err = ssh.ParsePrivateKey([]byte(key))
	} else {
		signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(key), []byte(password))
	}
	if err != nil {
		println(err.Error())
		return nil, err
	}
	return ssh.PublicKeys(signer), nil
}

// WithPrivateKeyTerminal 通过终端读取带密码的 PublicKey
func WithPrivateKeyTerminal(keyfile string) (ssh.AuthMethod, error) {
	// fmt.Fprintf(os.Stderr, "This SSH key is encrypted. Please enter passphrase for key '%s':", priv.path)
	passphrase, err := terminal.ReadPassword(int(syscall.Stdin))
	if err != nil {
		println(err.Error())
		return nil, err
	}

	fmt.Fprintln(os.Stderr)

	pemBytes, err := ioutil.ReadFile(keyfile)
	if err != nil {

		println(err.Error())
		return nil, err
	}
	signer, err := ssh.ParsePrivateKeyWithPassphrase(pemBytes, passphrase)
	if err != nil {

		fmt.Println(err)
		return nil, err
	}

	return ssh.PublicKeys(signer), nil
}
