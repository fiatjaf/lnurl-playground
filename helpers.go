package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/fiatjaf/go-lnurl"
	"github.com/lightningnetwork/lnd/lnwire"
	"github.com/lightningnetwork/lnd/zpay32"
)

type Preferences struct {
	Fail         bool
	MetadataSize int
}

const currency = "bc"

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz  ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randomLetter() string {
	return string(letterRunes[rand.Intn(len(letterRunes))])
}

var privkey, _ = btcec.NewPrivateKey(btcec.S256())
var hash, _ = hex.DecodeString("549cdb9911088e5c9c17569a60c920152610f6de79bf706c168565bfbd78b1bb")

func makeFakeInvoice(msat int, metadata string) string {
	var hash32 [32]byte
	for i := 0; i < 32; i++ {
		hash32[i] = hash[i]
	}

	descriptionhash := sha256.Sum256([]byte(metadata))
	var descriptionhash32 [32]byte
	for i := 0; i < 32; i++ {
		descriptionhash32[i] = descriptionhash[i]
	}

	invoice, _ := zpay32.NewInvoice(
		&chaincfg.Params{Bech32HRPSegwit: currency},
		hash32,
		time.Now(),
		zpay32.Destination(privkey.PubKey()),
		zpay32.DescriptionHash(descriptionhash32),
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
	if rand.Int63n(100) < 50 {
		fixed := rand.Int63n(1000) * 1000
		min = fixed
		max = fixed
	} else {
		min = rand.Int63n(1000) * 1000
		max = min * rand.Int63n(10)
	}

	return
}

func generateMetadata(size int) string {
	plain := ""
	for i := 0; i < size; i++ {
		plain += randomLetter()
	}

	metadata := [][]string{
		[]string{"text/plain", plain},
	}

	j, _ := json.Marshal(metadata)
	return string(j)
}

func randomSuccessAction() lnurl.SuccessAction {
	switch rand.Intn(3) {
	case 0:
		return lnurl.NoAction()
	case 1:
		return lnurl.Action(
			"You've paid!, now visit this URL: ",
			"https://lnurl.bigsun.xyz/")
	default: // case 2
		return lnurl.Action("Thanks!", "")
	}
}
