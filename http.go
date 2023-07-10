package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	log "github.com/cihub/seelog"
	"github.com/parnurzeal/gorequest"
	"github.com/raminhz90/esm/util"
	"github.com/valyala/fasthttp"
)

func BasicAuth(req *fasthttp.Request, user, pass string) {
	msg := fmt.Sprintf("%s:%s", user, pass)
	encoded := base64.StdEncoding.EncodeToString([]byte(msg))
	req.Header.Add("Authorization", "Basic "+encoded)
}

func Get(url string, auth *Auth, proxy string) (*http.Response, string, []error) {

	request := gorequest.New()

	tr := &http.Transport{
		DisableKeepAlives:  true,
		DisableCompression: false,
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	}
	request.Transport = tr

	if auth != nil {
		request.SetBasicAuth(auth.User, auth.Pass)
	}

	//request.Type("application/json")

	if len(proxy) > 0 {
		request.Proxy(proxy)
	}

	resp, body, errs := request.Get(url).End()
	return resp, body, errs

}

func Post(url string, auth *Auth, body string, proxy string) (*http.Response, string, []error) {
	request := gorequest.New()
	tr := &http.Transport{
		DisableKeepAlives:  true,
		DisableCompression: false,
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: true},
	}
	request.Transport = tr

	if auth != nil {
		request.SetBasicAuth(auth.User, auth.Pass)
	}

	//request.Type("application/json")

	if len(proxy) > 0 {
		request.Proxy(proxy)
	}

	request.Post(url)

	if len(body) > 0 {
		request.Send(body)
	}

	return request.End()
}

func newDeleteRequest(client *http.Client, method, urlStr string) (*http.Request, error) {
	if method == "" {
		// We document that "" means "GET" for Request.Method, and people have
		// relied on that from NewRequest, so keep that working.
		// We still enforce validMethod for non-empty methods.
		method = "GET"
	}
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	req := &http.Request{
		Method:     method,
		URL:        u,
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     make(http.Header),
		Host:       u.Host,
	}
	return req, nil
}

//
//func GzipHandler(req *http.Request) {
//	var b bytes.Buffer
//	var buf bytes.Buffer
//	g := gzip.NewWriter(&buf)
//
//	_, err := io.Copy(g, &b)
//	if err != nil {
//		panic(err)
//		//slog.Error(err)
//		return
//	}
//}

var client *http.Client = &http.Client{
	Transport: &http.Transport{
		DisableKeepAlives:  true,
		DisableCompression: false,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}
var fastHttpClient = &fasthttp.Client{
	TLSConfig: &tls.Config{InsecureSkipVerify: true},
}

func DoRequest(compress bool, method string, loadUrl string, auth *Auth, body []byte, proxy string) (string, error) {

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	//defer fasthttp.ReleaseRequest(req)   // <- do not forget to release
	//defer fasthttp.ReleaseResponse(resp) // <- do not forget to release

	req.SetRequestURI(loadUrl)
	req.Header.SetMethod(method)

	//req.Header.Set("Content-Type", "application/json")

	if compress {
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("content-encoding", "gzip")
	}

	if auth != nil {
		req.URI().SetUsername(auth.User)
		req.URI().SetPassword(auth.Pass)
	}

	if len(body) > 0 {

		//if compress {
		//	_, err := fasthttp.WriteGzipLevel(req.BodyWriter(), data.Bytes(), fasthttp.CompressBestSpeed)
		//	if err != nil {
		//		panic(err)
		//	}
		//} else {
		//	//req.SetBody(body)
		//	req.SetBodyStreamWriter(func(w *bufio.Writer) {
		//		w.Write(data.Bytes())
		//		w.Flush()
		//	})
		//
		//}

		if compress {
			_, err := fasthttp.WriteGzipLevel(req.BodyWriter(), body, fasthttp.CompressBestSpeed)
			if err != nil {
				panic(err)
			}
		} else {
			req.SetBody(body)

			//req.SetBodyStreamWriter(func(w *bufio.Writer) {
			//	w.Write(body)
			//	w.Flush()
			//})
		}
	}

	err := fastHttpClient.Do(req, resp)

	if err != nil {
		panic(err)
	}
	if resp == nil {
		panic("empty response")
	}

	if resp.StatusCode() == http.StatusOK || resp.StatusCode() == http.StatusCreated {
		log.Trace("received status code", resp.StatusCode, "from", string(resp.Header.Header()))

	} else {
		log.Error("received status code", resp.StatusCode, "from", string(resp.Header.Header()))
	}
	return string(resp.Body()), nil
}

func Request(method string, r string, auth *Auth, body *bytes.Buffer, proxy string) (string, error) {

	var err error
	var reqest *http.Request
	if body != nil {
		reqest, err = http.NewRequest(method, r, body)
	} else {
		reqest, err = newDeleteRequest(client, method, r)
	}

	if err != nil {
		panic(err)
	}

	if auth != nil {
		reqest.SetBasicAuth(auth.User, auth.Pass)
	}

	reqest.Header.Set("Content-Type", "application/json")

	resp, errs := client.Do(reqest)
	if errs != nil {
		log.Error(util.SubString(errs.Error(), 0, 500))
		return "", errs
	}

	if resp != nil && resp.Body != nil {

		defer resp.Body.Close()
	}

	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		return "", errors.New("server error: " + string(b))
	}

	respBody, err := io.ReadAll(resp.Body)

	log.Error(util.SubString(string(respBody), 0, 500))

	if err != nil {
		log.Error(util.SubString(string(err.Error()), 0, 500))
		return string(respBody), err
	}

	if err != nil {
		return string(respBody), err
	}
	io.Copy(io.Discard, resp.Body)
	defer resp.Body.Close()
	return string(respBody), nil
}

func DecodeJson(jsonStream string, o interface{}) error {

	decoder := json.NewDecoder(strings.NewReader(jsonStream))
	// UseNumber causes the Decoder to unmarshal a number into an interface{} as a Number instead of as a float64.
	decoder.UseNumber()
	//decoder.

	if err := decoder.Decode(o); err != nil {
		fmt.Println("error:", err)
		return err
	}
	return nil
}

func DecodeJsonBytes(jsonStream []byte, o interface{}) error {
	decoder := json.NewDecoder(bytes.NewReader(jsonStream))
	// UseNumber causes the Decoder to unmarshal a number into an interface{} as a Number instead of as a float64.
	decoder.UseNumber()

	if err := decoder.Decode(o); err != nil {
		fmt.Println("error:", err)
		return err
	}
	return nil
}
