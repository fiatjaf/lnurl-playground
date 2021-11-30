package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/fiatjaf/go-lnurl"
	lightning "github.com/fiatjaf/lightningd-gjson-rpc"
	"github.com/lightningnetwork/lnd/lnwire"
	"github.com/lightningnetwork/lnd/zpay32"
	"github.com/lucsky/cuid"
	"github.com/tidwall/gjson"
)

type Preferences struct {
	Disposable bool
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz  ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randomLetter() string {
	return string(letterRunes[rand.Intn(len(letterRunes))])
}

var privkey, _ = btcec.NewPrivateKey(btcec.S256())

func makeLNURLPayParams(session string) lnurl.LNURLPayParams {
	metadata := lnurl.Metadata{}
	metadata.Image.DataURI = "data:image/png;base64," + image
	for i := 0; i < rand.Intn(50); i++ {
		metadata.Description += randomLetter()
		metadata.LongDescription += randomLetter() + randomLetter()
	}

	min, max := generateMinMax()

	var commentAllowed int
	if rand.Intn(10) > 5 {
		commentAllowed = rand.Intn(15)
	}

	return lnurl.LNURLPayParams{
		Callback:        fmt.Sprintf("%s/lnurl-pay/callback/%s", s.ServiceURL, session),
		MinSendable:     min,
		MaxSendable:     max,
		Metadata:        metadata,
		EncodedMetadata: metadata.Encode(),
		Tag:             "payRequest",
		CommentAllowed:  int64(commentAllowed),
		PayerData: lnurl.PayerDataSpec{
			LightningAddress: &lnurl.PayerDataItemSpec{},
			Email:            &lnurl.PayerDataItemSpec{},
			FreeName:         &lnurl.PayerDataItemSpec{},
			PubKey:           &lnurl.PayerDataItemSpec{},
			KeyAuth:          &lnurl.PayerDataKeyAuthSpec{K1: lnurl.RandomK1()},
		},
	}
}

func makeInvoice(
	msat int64,
	params lnurl.LNURLPayParams,
	payerdata string,
) (string, []byte) {
	preimage, _ := hex.DecodeString(lnurl.RandomK1())

	var h [32]byte
	if payerdata == "" {
		h = params.HashMetadata()
	} else {
		h = params.HashWithPayerData(payerdata)
	}

	var bolt11 string
	var err error

	spark := &lightning.Client{
		SparkURL:    s.SparkoURL,
		SparkToken:  s.SparkoToken,
		CallTimeout: time.Second * 3,
	}
	var inv gjson.Result
	inv, err = spark.CallNamed("invoicewithdescriptionhash",
		"msatoshi", msat,
		"label", "lnurl.bigsun.xyz/"+cuid.Slug(),
		"description_hash", hex.EncodeToString(h[:]),
		"preimage", hex.EncodeToString(preimage),
	)
	bolt11 = inv.Get("bolt11").String()

	if err != nil {
		log.Warn().Err(err).Msg("couldn't generate real invoice, using a fake")
		return makeFakeInvoice(msat, h, preimage), preimage
	}
	return bolt11, preimage
}

func makeFakeInvoice(msat int64, h [32]byte, preimage []byte) string {
	hash := sha256.Sum256(preimage)

	invoice, _ := zpay32.NewInvoice(
		&chaincfg.Params{Bech32HRPSegwit: "bc"},
		hash,
		time.Now(),
		zpay32.Destination(privkey.PubKey()),
		zpay32.DescriptionHash(h),
		zpay32.Amount(lnwire.MilliSatoshi(msat)),
		zpay32.Expiry(time.Minute*60),
	)

	bolt11, _ := invoice.Encode(zpay32.MessageSigner{
		SignCompact: func(hash []byte) ([]byte, error) {
			return btcec.SignCompact(btcec.S256(),
				privkey, hash, true)
		},
	})

	return bolt11
}

func generateMinMax() (min, max int64) {
	if rand.Int63n(100) < 30 {
		fixed := (1 + rand.Int63n(15)) * 1000
		min = fixed
		max = fixed
	} else {
		min = (1 + rand.Int63n(5)) * 1000
		max = min * 4
	}

	return
}

func randomSuccessAction(
	preimage []byte,
	comment string,
	payer lnurl.PayerDataValues,
) *lnurl.SuccessAction {
	switch rand.Intn(3) {
	case 0:
		var message string

		switch {
		case payer.FreeName != "":
			message = fmt.Sprintf("Obrigado, %s! ", payer.FreeName)
		case payer.LightningAddress != "":
			message = fmt.Sprintf("Děkuji, %s! ", payer.LightningAddress)
		case payer.PubKey != "":
			message = fmt.Sprintf("Gracias, %s! ", payer.PubKey)
		case payer.KeyAuth != nil:
			message = fmt.Sprintf("Grazie, %s! ", payer.KeyAuth.Key)
		default:
			message = "Thank you! "
		}
		if comment != "" {
			message += fmt.Sprintf("You said: '%s'", comment)
		}

		return lnurl.Action(message, "")
	case 1:
		var message string

		if payer.PubKey != "" {
			message += fmt.Sprintf("Gracias, %s! ", payer.PubKey[:5])
		}
		if payer.LightningAddress != "" {
			message += fmt.Sprintf("Děkuji, %s! ", payer.LightningAddress)
		}
		if payer.KeyAuth != nil {
			message += fmt.Sprintf("Grazie, %s! ", payer.KeyAuth.Key[:5])
		}
		if payer.FreeName != "" {
			message += fmt.Sprintf("Obrigado, %s! ", payer.FreeName)
		}
		if comment != "" {
			message += fmt.Sprintf("You said: '%s'. ", comment)
		}

		message += "Here is your URL:"

		return lnurl.Action(message, "https://fiatjaf.com/")
	case 2:
		var name string

		switch {
		case payer.FreeName != "":
			name = payer.FreeName
		case payer.LightningAddress != "":
			name = payer.LightningAddress
		case payer.PubKey != "":
			name = payer.PubKey
		case payer.KeyAuth != nil:
			name = payer.KeyAuth.Key
		default:
			name = "Anonymous Donor"
		}

		var message string
		var secret string

		if comment == "" {
			message = "Your secret name is: "
			secret = name
		} else {
			message = fmt.Sprintf("Hello %s, your secret comment was: ", name)
			secret = comment
		}

		a, _ := lnurl.AESAction(message, preimage, secret)
		return a
	}

	return nil
}
