package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/elazarl/go-bindata-assetfs"
	"github.com/fiatjaf/go-lnurl"
	"gopkg.in/antage/eventsource.v1"
)

func setupHandlers() {
	http.HandleFunc("/set-preferences", func(w http.ResponseWriter, r *http.Request) {
		session := r.URL.Query().Get("session")
		mz, _ := strconv.Atoi(r.FormValue("metadata-size"))
		if mz == 0 {
			mz = 23
		}
		userParams[session] = Preferences{
			Fail:         r.FormValue("fail") != "false",
			MetadataSize: mz,
		}
		w.WriteHeader(200)
	})

	http.HandleFunc("/session", func(w http.ResponseWriter, r *http.Request) {
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

			go func() {
				time.Sleep(1 * time.Second)
				es.SendRetryMessage(3 * time.Second)
			}()

			go func() {
				for {
					time.Sleep(25 * time.Second)
					es.SendEventMessage("", "keepalive", "")
				}
			}()

			userStreams[session] = es
		}

		go func() {
			lnurllogin, _ := lnurl.LNURLEncode(fmt.Sprintf("%s/lnurl-login?tag=login&k1=%s", s.ServiceURL, session))
			lnurlwithdraw, _ := lnurl.LNURLEncode(fmt.Sprintf("%s/lnurl-withdraw?session=%s", s.ServiceURL, session))
			lnurlpay, _ := lnurl.LNURLEncode(fmt.Sprintf("%s/lnurl-pay?session=%s", s.ServiceURL, session))

			params, _ := json.Marshal(struct {
				LNURLLogin    string `json:"lnurllogin"`
				LNURLWithdraw string `json:"lnurlwithdraw"`
				LNURLPay      string `json:"lnurlpay"`
			}{lnurllogin, lnurlwithdraw, lnurlpay})
			es.SendEventMessage(string(params), "params", "")
		}()

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

	http.HandleFunc("/lnurl-withdraw", func(w http.ResponseWriter, r *http.Request) {
		session := r.URL.Query().Get("session")

		if p, ok := userParams[session]; ok && p.Fail && rand.Intn(10) < 3 {
			json.NewEncoder(w).Encode(lnurl.ErrorResponse("You asked for a FAILURE!"))
			return
		}

		min, max := generateMinMax()
		resp, _ := json.Marshal(lnurl.LNURLWithdrawResponse{
			LNURLResponse:      lnurl.LNURLResponse{Status: "OK"},
			Callback:           fmt.Sprintf("%s/lnurl-withdraw/callback/%s", s.ServiceURL, session),
			K1:                 lnurl.RandomK1(),
			MinWithdrawable:    min,
			MaxWithdrawable:    max,
			DefaultDescription: "sample withdraw",
			Tag:                "withdrawRequest",
		})

		if es, ok := userStreams[session]; ok {
			es.SendEventMessage(string(resp), "withdraw-req", "")
		}

		w.Write(resp)
	})

	http.HandleFunc("/lnurl-withdraw/callback/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		session := parts[len(parts)-1]

		if p, ok := userParams[session]; ok && p.Fail {
			json.NewEncoder(w).Encode(lnurl.ErrorResponse("You asked for a FAILURE!"))
			return
		}

		k1 := r.URL.Query().Get("k1")
		pr := r.URL.Query().Get("pr")
		json.NewEncoder(w).Encode(lnurl.OkResponse())

		if es, ok := userStreams[session]; ok {
			es.SendEventMessage(`{"invoice": "`+pr+`","k1":"`+k1+`"}`, "withdraw", "")
		}
	})

	http.HandleFunc("/lnurl-pay", func(w http.ResponseWriter, r *http.Request) {
		session := r.URL.Query().Get("session")

		if p, ok := userParams[session]; ok && p.Fail && rand.Intn(10) < 3 {
			json.NewEncoder(w).Encode(lnurl.ErrorResponse("You asked for a FAILURE!"))
			return
		}

		min, max := generateMinMax()

		var metadata string
		if p, ok := userParams[session]; ok && p.MetadataSize > 0 {
			metadata = generateMetadata(p.MetadataSize)
		} else {
			metadata = generateMetadata(23)
		}
		userMetadata[session] = metadata

		resp, _ := json.Marshal(lnurl.LNURLPayResponse1{
			LNURLResponse:   lnurl.LNURLResponse{Status: "OK"},
			Callback:        fmt.Sprintf("%s/lnurl-pay/callback/%s", s.ServiceURL, session),
			MinSendable:     min,
			MaxSendable:     max,
			EncodedMetadata: metadata,
			Tag:             "payRequest",
		})

		if es, ok := userStreams[session]; ok {
			es.SendEventMessage(string(resp), "pay_request", "")
		}

		w.Write(resp)
	})

	http.HandleFunc("/lnurl-pay/callback/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		session := parts[len(parts)-1]

		amount := r.URL.Query().Get("amount")
		fromnodes := r.URL.Query().Get("fromnodes")

		msat, err := strconv.Atoi(amount)
		if err != nil {
			json.NewEncoder(w).Encode(lnurl.ErrorResponse("amount is not integer"))
			return
		}

		if p, ok := userParams[session]; ok && p.Fail {
			json.NewEncoder(w).Encode(lnurl.ErrorResponse("You asked for a FAILURE!"))
			return
		}

		metadata, _ := userMetadata[session]
		delete(userMetadata, session)
		fakeinvoice := makeFakeInvoice(msat, metadata)

		resp, _ := json.Marshal(lnurl.LNURLPayResponse2{
			LNURLResponse: lnurl.LNURLResponse{Status: "OK"},
			PR:            fakeinvoice,
			SuccessAction: randomSuccessAction(),
			Routes:        make([][]lnurl.RouteInfo, 0),
		})

		if es, ok := userStreams[session]; ok {
			es.SendEventMessage(`{"fromnodes": "`+fromnodes+`","amount":"`+amount+`"}`, "pay", "")
			es.SendEventMessage(string(resp), "pay_result", "")
		}

		w.Write(resp)
	})

	http.Handle("/", http.FileServer(&assetfs.AssetFS{Asset: Asset, AssetDir: AssetDir, Prefix: "/static/"}))
}
