package actions

import (
	"bufio"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"fmt"
	"os"
	"strings"

	"github.com/go-acme/lego/v4/certificate"
	"github.com/go-acme/lego/v4/challenge/http01"
	"github.com/go-acme/lego/v4/lego"
	"github.com/go-acme/lego/v4/registration"
	"github.com/voidluo/trojan-go/cmd/trojan/menu"
)

type legoUser struct {
	Email        string
	Registration *registration.Resource
	key          *ecdsa.PrivateKey
}

func (u *legoUser) GetEmail() string                        { return u.Email }
func (u legoUser) GetRegistration() *registration.Resource { return u.Registration }
func (u *legoUser) GetPrivateKey() crypto.PrivateKey        { return u.key }

// getStdin 获取用户输入
func getStdin(promptCN, promptEN string) string {
	prompt := promptCN
	if menu.CurrentLang == menu.EN {
		prompt = promptEN
	}
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	s, _ := reader.ReadString('\n')
	return strings.TrimSpace(s)
}

// obtainCert 内部函数：执行证书申请，返回证书内容和私钥内容
func obtainCert(domain, email string) (*certificate.Resource, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}

	user := &legoUser{Email: email, key: privateKey}
	config := lego.NewConfig(user)
	config.CADirURL = lego.LEDirectoryProduction

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, err
	}

	provider := http01.NewProviderServer("", "80")
	if err = client.Challenge.SetHTTP01Provider(provider); err != nil {
		return nil, err
	}

	reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
	if err != nil {
		return nil, err
	}
	user.Registration = reg

	return client.Certificate.Obtain(certificate.ObtainRequest{
		Domains: []string{domain},
		Bundle:  true,
	})
}
