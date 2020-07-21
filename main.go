package main

import (
	"crypto/tls"
	"net/http"
	"os"

	"github.com/imroc/req"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
	"gopkg.in/antage/eventsource.v1"
)

type Settings struct {
	Port               string `envconfig:"PORT" required:"true"`
	ServiceURL         string `envconfig:"SERVICE_URL" required:"true"`
	SparkoURL          string `envconfig:"SPARKO_URL"`
	SparkoToken        string `envconfig:"SPARKO_TOKEN"`
	LndTestnetURL      string `envconfig:"LND_TESTNET_URL"`
	LndTestnetMacaroon string `envconfig:"LND_TESTNET_MACAROON"`
}

var err error
var s Settings
var log = zerolog.New(os.Stderr).Output(zerolog.ConsoleWriter{Out: os.Stderr})
var userStreams = make(map[string]eventsource.EventSource)
var userKeys = make(map[string]string)
var userParams = make(map[string]Preferences)
var userMetadata = make(map[string]string)

func main() {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	req.SetClient(http.DefaultClient)

	err = envconfig.Process("", &s)
	if err != nil {
		log.Fatal().Err(err).Msg("couldn't process envconfig.")
	}

	// routes
	setupHandlers()

	// start http server
	log.Print("listening at 0.0.0.0:" + s.Port)
	http.ListenAndServe("0.0.0.0:"+s.Port, nil)
}
