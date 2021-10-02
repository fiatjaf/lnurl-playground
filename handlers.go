package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

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
			Disposable:   r.FormValue("disposable") == "true",
			MetadataSize: mz,
			Currency:     r.FormValue("currency"),
		}
		w.WriteHeader(200)
	})

	http.HandleFunc("/trigger-notify", func(w http.ResponseWriter, r *http.Request) {
		notifyURL := r.FormValue("notifyURL")
		client := http.Client{Timeout: 3 * time.Second}
		go client.Post(notifyURL, "", nil)
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
				time.Sleep(2 * time.Second)
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
			time.Sleep(400 * time.Millisecond)
			lnurllogin, _ := lnurl.LNURLEncode(fmt.Sprintf("%s/lnurl-login?tag=login&k1=%s", s.ServiceURL, session))
			lnurlwithdraw, _ := lnurl.LNURLEncode(fmt.Sprintf("%s/lnurl-withdraw?session=%s", s.ServiceURL, session))
			lnurlpay, _ := lnurl.LNURLEncode(fmt.Sprintf("%s/lnurl-pay?session=%s", s.ServiceURL, session))
			lnurlchannel, _ := lnurl.LNURLEncode(fmt.Sprintf("%s/lnurl-channel?session=%s", s.ServiceURL, session))

			params, _ := json.Marshal(struct {
				LNURLLogin    string `json:"lnurllogin"`
				LNURLWithdraw string `json:"lnurlwithdraw"`
				LNURLPay      string `json:"lnurlpay"`
				LNURLChannel  string `json:"lnurlchannel"`
			}{lnurllogin, lnurlwithdraw, lnurlpay, lnurlchannel})
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

		if p, ok := userParams[session]; ok && p.Fail {
			json.NewEncoder(w).Encode(lnurl.ErrorResponse("You asked for a FAILURE!"))
			return
		}

		min, max := generateMinMax()
		resp, _ := json.Marshal(lnurl.LNURLWithdrawResponse{
			Tag: "withdrawRequest",
			Callback: fmt.Sprintf(
				"%s/lnurl-withdraw/callback/%s", s.ServiceURL, session),
			K1:                 lnurl.RandomK1(),
			MinWithdrawable:    min,
			MaxWithdrawable:    max,
			DefaultDescription: "sample withdraw",
			BalanceCheck:       fmt.Sprintf("%s/lnurl-withdraw?session=%s", s.ServiceURL, session),
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
		balanceNotify := r.URL.Query().Get("balanceNotify")

		json.NewEncoder(w).Encode(lnurl.OkResponse())

		if es, ok := userStreams[session]; ok {
			es.SendEventMessage(`{"pr": "`+pr+`","k1":"`+k1+`","balanceNotify": "`+balanceNotify+`"}`, "withdraw", "")
		}
	})

	http.HandleFunc("/lnurl-channel", func(w http.ResponseWriter, r *http.Request) {
		session := r.URL.Query().Get("session")

		if p, ok := userParams[session]; ok && p.Fail {
			json.NewEncoder(w).Encode(lnurl.ErrorResponse("You asked for a FAILURE!"))
			return
		}

		resp, _ := json.Marshal(lnurl.LNURLChannelResponse{
			Callback: fmt.Sprintf("%s/lnurl-channel/callback/%s", s.ServiceURL, session),
			K1:       lnurl.RandomK1(),
			Tag:      "channelRequest",
			URI:      "0331f80652fb840239df8dc99205792bba2e559a05469915804c08420230e23c7c@74.108.13.152:9735",
		})

		if es, ok := userStreams[session]; ok {
			es.SendEventMessage(string(resp), "channel-req", "")
		}

		w.Write(resp)
	})

	http.HandleFunc("/lnurl-channel/callback/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		session := parts[len(parts)-1]

		if p, ok := userParams[session]; ok && p.Fail {
			json.NewEncoder(w).Encode(lnurl.ErrorResponse("You asked for a FAILURE!"))
			return
		}

		k1 := r.URL.Query().Get("k1")
		private := r.URL.Query().Get("private")
		remoteid := r.URL.Query().Get("remoteid")
		json.NewEncoder(w).Encode(lnurl.OkResponse())

		if es, ok := userStreams[session]; ok {
			es.SendEventMessage(`{"private": "`+private+`", "remoteid": "`+remoteid+`", "k1":"`+k1+`"}`, "channel", "")
		}
	})

	http.HandleFunc("/lnurl-pay", func(w http.ResponseWriter, r *http.Request) {
		session := r.URL.Query().Get("session")

		if p, ok := userParams[session]; ok && p.Fail && rand.Intn(10) < 3 {
			json.NewEncoder(w).Encode(lnurl.ErrorResponse("You asked for a FAILURE!"))
			return
		}

		min, max := generateMinMax()

		var metadata lnurl.Metadata
		if p, ok := userParams[session]; ok && p.MetadataSize > 0 {
			metadata = generateMetadata(p.MetadataSize)
		} else {
			metadata = generateMetadata(23)
		}
		userMetadata[session] = metadata

		resp, _ := json.Marshal(lnurl.LNURLPayParams{
			Callback:       fmt.Sprintf("%s/lnurl-pay/callback/%s", s.ServiceURL, session),
			MinSendable:    min,
			MaxSendable:    max,
			Metadata:       metadata,
			Tag:            "payRequest",
			CommentAllowed: 8,
			PayerData: lnurl.PayerDataSpec{
				LightningAddress: &lnurl.PayerDataItemSpec{},
				Email:            &lnurl.PayerDataItemSpec{},
				FreeName:         &lnurl.PayerDataItemSpec{},
				PubKey:           &lnurl.PayerDataItemSpec{},
				KeyAuth:          &lnurl.PayerDataKeyAuthSpec{K1: lnurl.RandomK1()},
			},
		})

		if es, ok := userStreams[session]; ok {
			es.SendEventMessage(string(resp), "pay-req", "")
		}

		w.Write(resp)
	})

	http.HandleFunc("/lnurl-pay/callback/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(r.URL.Path, "/")
		session := parts[len(parts)-1]

		amount := r.URL.Query().Get("amount")
		comment := r.URL.Query().Get("comment")
		payerdata := r.URL.Query().Get("payerdata")

		msat, err := strconv.ParseInt(amount, 10, 64)
		if err != nil {
			json.NewEncoder(w).Encode(lnurl.ErrorResponse("amount is not integer"))
			return
		}

		var currency = "bc"
		var disposable = lnurl.TRUE
		if p, ok := userParams[session]; ok {
			if p.Fail {
				json.NewEncoder(w).Encode(lnurl.ErrorResponse("You asked for a FAILURE!"))
				return
			}

			if p.Currency != "" {
				currency = p.Currency
			}

			if p.Disposable == false {
				disposable = lnurl.FALSE
			}
		}

		metadata, _ := userMetadata[session]
		delete(userMetadata, session)
		bolt11, preimage := makeInvoice(msat, currency, metadata, payerdata)

		var payerData lnurl.PayerDataValues
		json.Unmarshal([]byte(payerdata), &payerData)

		resp, _ := json.Marshal(lnurl.LNURLPayValues{
			PR:            bolt11,
			SuccessAction: randomSuccessAction(preimage, comment, payerData),
			Routes:        []struct{}{},
			Disposable:    disposable,
		})

		if es, ok := userStreams[session]; ok {
			j, _ := json.Marshal(struct {
				Amount    string `json:"amount"`
				Comment   string `json:"comment,omitempty"`
				PayerData string `json:"payerdata,omitempty"`
			}{amount, comment, payerdata})

			es.SendEventMessage(string(j), "pay", "")
			es.SendEventMessage(string(resp), "pay_result", "")
		}

		w.Write(resp)
	})

	// serve static client
	if staticFS, err := fs.Sub(static, "static"); err != nil {
		log.Fatal().Err(err).Msg("failed to load static files subdir")
		return
	} else {
		spaFS := SpaFS{staticFS}
		httpFS := http.FS(spaFS)
		http.Handle("/", http.FileServer(httpFS))
	}
}

type SpaFS struct {
	base fs.FS
}

func (s SpaFS) Open(name string) (fs.File, error) {
	if file, err := s.base.Open(name); err == nil {
		return file, nil
	} else {
		return s.base.Open("index.html")
	}
}
