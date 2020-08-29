package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/fiatjaf/go-lnurl"
	lightning "github.com/fiatjaf/lightningd-gjson-rpc"
	"github.com/imroc/req"
	"github.com/lightningnetwork/lnd/lnwire"
	"github.com/lightningnetwork/lnd/zpay32"
	"github.com/lucsky/cuid"
	"github.com/tidwall/gjson"
)

type Preferences struct {
	Fail         bool
	Disposable   bool
	MetadataSize int
	Currency     string
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyz  ABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randomLetter() string {
	return string(letterRunes[rand.Intn(len(letterRunes))])
}

var privkey, _ = btcec.NewPrivateKey(btcec.S256())

func makeInvoice(msat int64, currency string, metadata string) (string, []byte) {
	preimage, _ := hex.DecodeString(lnurl.RandomK1())
	h := sha256.Sum256([]byte(metadata))

	var bolt11 string
	var err error

	switch currency {
	case "bc":
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
	case "tb":
		r, werr := req.Post(s.LndTestnetURL+"/v1/invoices", req.Header{
			"Grpc-Metadata-macaroon": s.LndTestnetMacaroon,
		}, req.BodyJSON(struct {
			ValueMsat       int64  `json:"value_msat"`
			DescriptionHash string `json:"description_hash"`
			RPreimage       string `json:"r_preimage"`
		}{
			msat,
			base64.StdEncoding.EncodeToString(h[:]),
			base64.StdEncoding.EncodeToString(preimage),
		}))
		if werr != nil {
			err = werr
			break
		}
		bolt11 = gjson.Parse(r.String()).Get("payment_request").String()
	}

	if err != nil {
		log.Warn().Err(err).Msg("couldn't generate real invoice, using a fake")
		return makeFakeInvoice(msat, currency, h, preimage), preimage
	}
	return bolt11, preimage
}

func makeFakeInvoice(msat int64, currency string, h [32]byte, preimage []byte) string {
	hash := sha256.Sum256(preimage)

	invoice, _ := zpay32.NewInvoice(
		&chaincfg.Params{Bech32HRPSegwit: currency},
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

func generateMetadata(size int) string {
	plain := ""
	for i := 0; i < size; i++ {
		plain += randomLetter()
	}

	metadata := [][]string{
		[]string{"text/plain", plain},
		[]string{"image/png;base64", image},
	}

	j, _ := json.Marshal(metadata)
	return string(j)
}

func randomSuccessAction(preimage []byte) *lnurl.SuccessAction {
	switch rand.Intn(3) {
	case 0:
		return nil
	case 1:
		return lnurl.Action(
			"You've paid!, now visit this URL: ",
			"https://lnurl.bigsun.xyz/")
	case 2:
		a, _ := lnurl.AESAction("You've paid, here's your code: ", preimage, "1234")
		return a
	default: // case 4
		return lnurl.Action("Thanks!", "")
	}
}
