package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/elazarl/go-bindata-assetfs"
	"github.com/fiatjaf/go-lnurl"
	"gopkg.in/antage/eventsource.v1"
)

func setupHandlers() {
	http.HandleFunc("/lnurl-withdraw", func(w http.ResponseWriter, r *http.Request) {
		session := r.URL.Query().Get("session")

		json.NewEncoder(w).Encode(lnurl.LNURLWithdrawResponse{
			LNURLResponse:      lnurl.LNURLResponse{Status: "OK"},
			Callback:           fmt.Sprintf("%s/lnurl-withdraw/callback/%s", s.ServiceURL, session),
			K1:                 randomHex(64), // use a new k1 here just because we can
			MaxWithdrawable:    4700123,
			MinWithdrawable:    444000,
			DefaultDescription: "sample withdraw",
			Tag:                "withdrawRequest",
		})
	})

	http.HandleFunc("/lnurl-withdraw/callback/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		session := parts[len(parts)-1]
		pubkey := userKeys[session]

		k1 := r.URL.Query().Get("k1")
		sig := r.URL.Query().Get("sig")
		pr := r.URL.Query().Get("pr")

		valid := "yes"
		if ok, err := lnurl.VerifySignature(k1, sig, pubkey); !ok || err != nil {
			json.NewEncoder(w).Encode(lnurl.ErrorResponse("Invalid signature!"))
			valid = "no"
		} else {
			json.NewEncoder(w).Encode(lnurl.OkResponse())
		}

		if es, ok := userStreams[session]; ok {
			es.SendEventMessage(`{"invoice": "`+pr+`","k1":"`+k1+`","sig":"`+sig+`","valid":"`+valid+`"}`, "withdraw", "")
		}
	})

	http.HandleFunc("/get-params", func(w http.ResponseWriter, r *http.Request) {
		session := randomHex(64)
		lnurllogin, _ := lnurl.LNURLEncode(fmt.Sprintf("%s/lnurl-login?tag=login&k1=%s", s.ServiceURL, session))
		lnurlwithdraw, _ := lnurl.LNURLEncode(fmt.Sprintf("%s/lnurl-withdraw?session=%s", s.ServiceURL, session))

		w.Header().Add("Content-Type", "application/json")
		json.NewEncoder(w).Encode(struct {
			Session            string `json:"session"`
			LNURLLoginLogin    string `json:"lnurllogin"`
			LNURLLoginWithdraw string `json:"lnurlwithdraw"`
		}{session, lnurllogin, lnurlwithdraw})
	})

	http.HandleFunc("/user-data", func(w http.ResponseWriter, r *http.Request) {
		session := r.URL.Query().Get("session")
		es, ok := userStreams[session]

		if !ok {
			es = eventsource.New(
				eventsource.DefaultSettings(),
				func(r *http.Request) [][]byte {
					return [][]byte{
						[]byte("X-Accel-Buffering: no"),
						[]byte("Cache-Control: no-cache"),
						[]byte("Content-Type: text/event-stream"),
						[]byte("Connection: keep-alive"),
						[]byte("Access-Control-Allow-Origin: *"),
					}
				},
			)

			userStreams[session] = es
		}

		es.ServeHTTP(w, r)
	})

	http.HandleFunc("/lnurl-login", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.String(), "?")
		actualQS := parts[len(parts)-1] // last ? segment
		params, err := url.ParseQuery(actualQS)
		if err != nil {
			log.Print("borked querystring " + r.URL.String() + ": " + err.Error())
			return
		}

		k1 := params.Get("k1")
		sig := params.Get("sig")
		key := params.Get("key")

		if ok, err := lnurl.VerifySignature(k1, sig, key); !ok {
			log.Warn().Err(err).Msg("initial signature verification failed")
			return
		}

		session := k1
		log.Debug().Str("session", session).Str("pubkey", key).Msg("valid login")
		userKeys[session] = key

		// if there's an active login SSE stream for this, notify there
		if es, ok := userStreams[session]; ok {
			es.SendEventMessage(`{"key":"`+key+`","k1":"`+k1+`","sig":"`+sig+`"}`, "login", "")
		}

		json.NewEncoder(w).Encode(lnurl.LNURLResponse{Status: "OK"})
	})

	http.Handle("/", http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: "/static/"}))
}
