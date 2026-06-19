package scenario

import (
	"bytes"
	"fmt"
	_ "net/http/pprof"
	"os"
	"sync"
	"time"

	netproxy "golang.org/x/net/proxy"

	_ "github.com/voidluo/trojan-go/api"
	_ "github.com/voidluo/trojan-go/api/service"
	"github.com/voidluo/trojan-go/common"
	_ "github.com/voidluo/trojan-go/component"
	_ "github.com/voidluo/trojan-go/log/golog"
	"github.com/voidluo/trojan-go/proxy"
	_ "github.com/voidluo/trojan-go/proxy/server"
	_ "github.com/voidluo/trojan-go/statistic/memory"
	"github.com/voidluo/trojan-go/test/util"
)

// test key and cert

var cert = `
-----BEGIN CERTIFICATE-----
MIIC5TCCAc2gAwIBAgIJAJqNVe6g/10vMA0GCSqGSIb3DQEBCwUAMBQxEjAQBgNV
BAMMCWxvY2FsaG9zdDAeFw0yMTA5MTQwNjE1MTFaFw0yNjA5MTMwNjE1MTFaMBQx
EjAQBgNVBAMMCWxvY2FsaG9zdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC
ggEBAK7bupJ8tmHM3shQ/7N730jzpRsXdNiBxq/Jxx8j+vB3AcxuP5bjXQZqS6YR
5W5vrfLlegtq1E/mmaI3Ht0RfIlzev04Dua9PWmIQJD801nEPknbfgCLXDh+pYr2
sfg8mUh3LjGtrxyH+nmbTjWg7iWSKohmZ8nUDcX94Llo5FxibMAz8OsAwOmUueCH
jP3XswZYHEy+OOP3K0ZEiJy0f5T6ZXk9OWYuPN4VQKJx1qrc9KzZtSPHwqVdkGUi
ase9tOPA4aMutzt0btgW7h7UrvG6C1c/Rr1BxdiYq1EQ+yypnAlyToVQSNbo67zz
wGQk4GeruIkOgJOLdooN/HjhbHMCAwEAAaM6MDgwFAYDVR0RBA0wC4IJbG9jYWxo
b3N0MAsGA1UdDwQEAwIHgDATBgNVHSUEDDAKBggrBgEFBQcDATANBgkqhkiG9w0B
AQsFAAOCAQEASsBzHHYiWDDiBVWUEwVZAduTrslTLNOxG0QHBKsHWIlz/3QlhQil
ywb3OhfMTUR1dMGY5Iq5432QiCHO4IMCOv7tDIkgb4Bc3v/3CRlBlnurtAmUfNJ6
pTRSlK4AjWpGHAEEd/8aCaOE86hMP8WDht8MkJTRrQqpJ1HeDISoKt9nepHOIsj+
I2zLZZtw0pg7FuR4MzWuqOt071iRS46Pupryb3ZEGIWNz5iLrDQod5Iz2ZGSRGqE
rB8idX0mlj5AHRRanVR3PAes+eApsW9JvYG/ImuCOs+ZsukY614zQZdR+SyFm85G
4NICyeQsmiypNHHgw+xZmGqZg65bXNGoyg==
-----END CERTIFICATE-----
`

var key = `
-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCu27qSfLZhzN7I
UP+ze99I86UbF3TYgcavyccfI/rwdwHMbj+W410GakumEeVub63y5XoLatRP5pmi
Nx7dEXyJc3r9OA7mvT1piECQ/NNZxD5J234Ai1w4fqWK9rH4PJlIdy4xra8ch/p5
m041oO4lkiqIZmfJ1A3F/eC5aORcYmzAM/DrAMDplLngh4z917MGWBxMvjjj9ytG
RIictH+U+mV5PTlmLjzeFUCicdaq3PSs2bUjx8KlXZBlImrHvbTjwOGjLrc7dG7Y
Fu4e1K7xugtXP0a9QcXYmKtREPssqZwJck6FUEjW6Ou888BkJOBnq7iJDoCTi3aK
Dfx44WxzAgMBAAECggEBAKYhib/H0ZhWB4yWuHqUxG4RXtrAjHlvw5Acy5zgmHiC
+Sh7ztrTJf0EXN9pvWwRm1ldgXj7hMBtPaaLbD1pccM9/qo66p17Sq/LjlyyeTOe
affOHIbz4Sij2zCOdkR9fr0EztTQScF3yBhl4Aa/4cO8fcCeWxm86WEldq9x4xWJ
s5WMR4CnrOJhDINLNPQPKX92KyxEQ/RfuBWovx3M0nl3fcUWfESY134t5g/UBFId
In19tZ+pGIpCkxP0U1AZWrlZRA8Q/3sO2orUpoAOdCrGk/DcCTMh0c1pMzbYZ1/i
cYXn38MpUo8QeG4FElUhAv6kzeBIl2tRBMVzIigo+AECgYEA3No1rHdFu6Ox9vC8
E93PTZevYVcL5J5yx6x7khCaOLKKuRXpjOX/h3Ll+hlN2DVAg5Jli/JVGCco4GeK
kbFLSyxG1+E63JbgsVpaEOgvFT3bHHSPSRJDnIU+WkcNQ2u4Ky5ahZzbNdV+4fj2
NO2iMgkm7hoJANrm3IqqW8epenMCgYEAyq+qdNj5DiDzBcDvLwY+4/QmMOOgDqeh
/TzhbDRyr+m4xNT7LLS4s/3wcbkQC33zhMUI3YvOHnYq5Ze/iL/TSloj0QCp1I7L
J7sZeM1XimMBQIpCfOC7lf4tU76Fz0DTHAL+CmX1DgmRJdYO09843VsKkscC968R
4cwL5oGxxgECgYAM4TTsH/CTJtLEIfn19qOWVNhHhvoMlSkAeBCkzg8Qa2knrh12
uBsU3SCIW11s1H40rh758GICDJaXr7InGP3ZHnXrNRlnr+zeqvRBtCi6xma23B1X
F5eV0zd1sFsXqXqOGh/xVtp54z+JEinZoForLNl2XVJVGG8KQZP50kUR/QKBgH4O
8zzpFT0sUPlrHVdp0wODfZ06dPmoWJ9flfPuSsYN3tTMgcs0Owv3C+wu5UPAegxB
X1oq8W8Qn21cC8vJQmgj19LNTtLcXI3BV/5B+Aghu02gr+lq/EA1bYuAG0jjUGlD
kyx0bQzl9lhJ4b70PjGtxc2z6KyTPdPpTB143FABAoGAQDoIUdc77/IWcjzcaXeJ
8abak5rAZA7cu2g2NVfs+Km+njsB0pbTwMnV1zGoFABdaHLdqbthLWtX7WOb1PDD
MQ+kbiLw5uj8IY2HEqJhDGGEdXBqxbW7kyuIAN9Mw+mwKzkikNcFQdxgchWH1d1o
lVkr92iEX+IhIeYb4DN1vQw=
-----END PRIVATE KEY-----
`

func init() {
	os.WriteFile("server.crt", []byte(cert), 0o777)
	os.WriteFile("server.key", []byte(key), 0o777)
}

func CheckClientServer(clientData, serverData string, socksPort int) (ok bool) {
	server, err := proxy.NewProxyFromConfigData([]byte(serverData), false)
	common.Must(err)
	go server.Run()

	client, err := proxy.NewProxyFromConfigData([]byte(clientData), false)
	common.Must(err)
	go client.Run()

	time.Sleep(time.Second * 2)
	dialer, err := netproxy.SOCKS5("tcp", fmt.Sprintf("127.0.0.1:%d", socksPort), nil, netproxy.Direct)
	common.Must(err)

	ok = true
	const num = 100
	wg := sync.WaitGroup{}
	wg.Add(num)
	for i := 0; i < num; i++ {
		go func() {
			const payloadSize = 1024
			payload := util.GeneratePayload(payloadSize)
			buf := [payloadSize]byte{}

			conn, err := dialer.Dial("tcp", util.EchoAddr)
			common.Must(err)

			common.Must2(conn.Write(payload))
			common.Must2(conn.Read(buf[:]))

			if !bytes.Equal(payload, buf[:]) {
				ok = false
			}
			conn.Close()
			wg.Done()
		}()
	}
	wg.Wait()
	client.Close()
	server.Close()
	return
}
