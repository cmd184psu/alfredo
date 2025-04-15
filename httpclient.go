package alfredo

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
	"strconv"
	"strings"
	"time"
)

type HttpMethod string

const (
	HttpGET     HttpMethod = "GET"
	HttpPOST    HttpMethod = "POST"
	HttpPUT     HttpMethod = "PUT"
	HttpDELETE  HttpMethod = "DELETE"
	HttpHEAD    HttpMethod = "HEAD"
	HttpOPTIONS HttpMethod = "OPTIONS"
	HttpPATCH   HttpMethod = "PATCH"
)

const (
	HEADER_AUTHORIZATION = "Authorization"
	HEADER_CONTENT_TYPE  = "Content-Type"
	HEADER_ACCEPT        = "Accept"
	CONTENT_TYPE_JSON    = "application/json"
)

const (
	httpapi_default_timeout = 30
)

type HttpApiStruct struct {
	UserName string `json:"userName"`
	Password string `json:"password"`

	Fqdn           string `json:"fqdn"`
	responseBody   []byte
	requestPayload []byte
	statusCode     int
	ssh            SSHStruct
	Timeout        int               `json:"timeout"`
	QueryParams    map[string]string `json:"queryParams"`
	Headers        map[string]string `json:"headers"`
	forceLocal     bool
	forceRemote    bool
	Secure         bool `json:"secure"`
	token          string
	Port           int    `json:"port"`
	IgnoreConflict bool   `json:"ignoreConflict"`
	Passcode       string `json:"passcode"`
}

func (has *HttpApiStruct) Load(filename string) error {
	if FileExistsEasy(filename) {
		if err := ReadStructFromJSONFile(filename, &has); err != nil {
			return err
		}
	} else {
		jsonContent := "[]"
		json.Unmarshal([]byte(jsonContent), &has)
	}
	return nil
}

func (has HttpApiStruct) HasFqdn() bool {
	return len(has.Fqdn) > 0
}

func (has *HttpApiStruct) SetFqdn(fqdn string) {
	has.Fqdn = fqdn
}
func (has HttpApiStruct) WithFqdn(fqdn string) HttpApiStruct {
	has.Fqdn = fqdn
	return has
}

func (has *HttpApiStruct) SetSSH(s SSHStruct) {
	has.ssh = s
	//has.sshEnabled = true
	has.Fqdn = s.Host
}
func (has HttpApiStruct) GetSSHEnabled() bool {
	return len(has.ssh.Host) > 0
}
func (has HttpApiStruct) GetSSH() SSHStruct {
	return has.ssh
}
func (has HttpApiStruct) WithSSH(s SSHStruct) HttpApiStruct {
	has.SetSSH(s)
	return has
}
func (has *HttpApiStruct) SetPayload(p []byte) {
	has.requestPayload = p
}

func (has *HttpApiStruct) SetPayloadAny(v any) error {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("Error marshalling JSON: %v", err)
	}
	has.SetPayload(jsonBytes)
	return nil
}

func (has HttpApiStruct) WithPayload(p []byte) HttpApiStruct {
	has.SetPayload(p)
	return has
}

func (has HttpApiStruct) WithPayloadAny(v any) HttpApiStruct {
	if err := has.SetPayloadAny(v); err != nil {
		panic(err)
	}
	return has
}

func (has HttpApiStruct) GetPayload() []byte {
	return has.requestPayload
}

func (has HttpApiStruct) IsPayloadEmpty() bool {
	if has.requestPayload == nil {
		return true
	}
	return len(has.requestPayload) == 0
}

func (has *HttpApiStruct) SetResponseBody(b []byte) {
	has.responseBody = b
}

func (has HttpApiStruct) WithResponseBody(b []byte) HttpApiStruct {
	has.SetResponseBody(b)
	return has
}

func (has HttpApiStruct) GetResponseBody() []byte {
	return has.responseBody
}

func (has *HttpApiStruct) SetStatusCode(s int) {
	has.statusCode = s
}

func (has HttpApiStruct) WithStatusCode(s int) HttpApiStruct {
	has.SetStatusCode(s)
	return has
}

func (has HttpApiStruct) GetStatusCode() int {
	return has.statusCode
}

func (has *HttpApiStruct) SetTimeout(t int) {
	has.Timeout = t
}

func (has HttpApiStruct) WithTimeout(t int) HttpApiStruct {
	has.SetTimeout(t)
	return has
}

func (has HttpApiStruct) GetTimeout() int {
	return has.Timeout
}

func (has *HttpApiStruct) SetForceLocal(b bool) {
	has.forceLocal = b
}

func (has *HttpApiStruct) GetForceLocal() bool {
	return has.forceLocal
}

func (has HttpApiStruct) GetPort() int {
	return has.Port
}
func (has *HttpApiStruct) SetPort(p int) {
	has.Port = p
}

func (has HttpApiStruct) WithPort(p int) HttpApiStruct {
	has.SetPort(p)
	return has
}

func (has *HttpApiStruct) SetQueryParams(qp map[string]string) {
	has.QueryParams = qp
}

func (has *HttpApiStruct) SetQueryParamsByURI(uri string) {
	qp := make(map[string]string)
	if strings.Contains(uri, "?") {
		uri = uri[strings.Index(uri, "?")+1:]
	}
	if len(uri) > 0 {
		uri = strings.ReplaceAll(uri, "&", " ") // replace all & with space
	}

	uri = strings.TrimSpace(uri)
	if len(uri) > 0 {
		uriList := strings.Split(uri, " ")
		for i := 0; i < len(uriList); i++ {
			if strings.Contains(uriList[i], "=") {
				nameval := strings.Split(uriList[i], "=")
				qp[nameval[0]] = nameval[1]
			}
		}
	}
	has.QueryParams = qp
}

func (has HttpApiStruct) WithQueryParamsByURI(uri string) HttpApiStruct {
	has.SetQueryParamsByURI(uri)
	return has
}

func (has HttpApiStruct) GetQueryParamsAsURI() string {
	var result string
	for k, v := range has.QueryParams {
		result = fmt.Sprintf("%s&%s=%s", result, k, v)
	}
	return result
}
func (has HttpApiStruct) GetQueryParamsAsCurlParams() string {
	var result string
	for k, v := range has.QueryParams {
		result = fmt.Sprintf("%s --data-urlencode %s=%s", result, k, v)
	}
	return result
}

func (has HttpApiStruct) WithQueryParams(qp map[string]string) HttpApiStruct {
	has.SetQueryParams(qp)
	return has
}

func (has HttpApiStruct) GetQueryParams() map[string]string {
	return has.QueryParams
}

func (has *HttpApiStruct) SetQueryPair(n string, v string) {
	if has.QueryParams == nil {
		has.QueryParams = make(map[string]string)
	}
	has.QueryParams[n] = v
}

func (has *HttpApiStruct) SetQueryPairInt(n string, v int) {
	if has.QueryParams == nil {
		has.QueryParams = make(map[string]string)
	}
	has.QueryParams[n] = strconv.Itoa(v)
}

func (has *HttpApiStruct) SetQueryPairBool(n string, v bool) {
	if has.QueryParams == nil {
		has.QueryParams = make(map[string]string)
	}
	has.QueryParams[n] = strconv.FormatBool(v)
}

func (has *HttpApiStruct) SetQueryPairFloat(n string, v float64) {
	if has.QueryParams == nil {
		has.QueryParams = make(map[string]string)
	}
	has.QueryParams[n] = strconv.FormatFloat(v, 'f', -1, 64)
}

func (has *HttpApiStruct) SetQueryPairInt64(n string, v int64) {
	if has.QueryParams == nil {
		has.QueryParams = make(map[string]string)
	}
	has.QueryParams[n] = strconv.FormatInt(v, 10)
}

func (has *HttpApiStruct) SetHeaders(h map[string]string) {
	has.Headers = h
}

func (has HttpApiStruct) WithHeaders(h map[string]string) HttpApiStruct {
	has.SetHeaders(h)
	return has
}

func (has HttpApiStruct) GetHeaders() map[string]string {
	return has.Headers
}
func (has HttpApiStruct) GetHeader(n string) string {
	if has.Headers == nil {
		return ""
	}

	if _, ok := has.Headers[n]; !ok {
		return ""
	}
	return has.Headers[n]
}

func (has *HttpApiStruct) SetHeader(n string, v string) {
	if has.Headers == nil {
		has.Headers = make(map[string]string)
	}
	has.Headers[n] = v
}
func (has *HttpApiStruct) HasHeader(n string) bool {
	_, ok := has.Headers[n]
	return ok
}

func (has *HttpApiStruct) SetAuthorizationHeader(v string) {
	has.SetHeader(HEADER_AUTHORIZATION, v)
}

func (has *HttpApiStruct) SetContentTypeHeader(v string) {
	has.SetHeader(HEADER_CONTENT_TYPE, v)
}

func (has *HttpApiStruct) SetAcceptHeader(v string) {
	has.SetHeader(HEADER_ACCEPT, v)
}

func (has *HttpApiStruct) SetContentTypeHeaderJSON() {
	has.SetContentTypeHeader(CONTENT_TYPE_JSON)
}

func routeToDataEncodedQuery(uri string) string {
	var result string
	namevals := strings.Split(uri, "&")

	for i := 0; i < len(namevals); i++ {
		temp := fmt.Sprintf("%s --data-urlencode %s", result, namevals[i])
		result = temp
	}
	return result
}

func (has HttpApiStruct) GetProtocol() string {
	if has.Secure {
		return "https"
	}
	return "http"
}

func (has HttpApiStruct) getBaseURL() string {
	cli := fmt.Sprintf("%s://%s", has.GetProtocol(), has.Fqdn)

	if has.Port == 0 {
		panic("port is 0")
	}

	if !(has.Port == 80 && strings.EqualFold(cli[0:7], "http://")) && !(has.Port == 443 && strings.EqualFold(cli[0:8], "https://")) {
		cli = fmt.Sprintf("%s:%d", cli, has.Port)
	}

	return cli
}

func (has HttpApiStruct) BuildCurlCLI(method string, uri string) string {
	VerbosePrintf("BEGIN HttpApiStruct::BuildCurlCLI(%s,%s)", method, uri)
	var realroute string
	if has.Timeout == 0 {
		has.Timeout = httpapi_default_timeout
	}
	cli := fmt.Sprintf("curl -m %d -ss -k -X %s", has.Timeout, method)
	if !has.IsPayloadEmpty() && !strings.EqualFold(method, string(HttpGET)) && !strings.EqualFold(method, string(HttpHEAD)) {
		cli = fmt.Sprintf("%s -d @-", cli)
	}
	if len(has.UserName) > 0 && len(has.Password) > 0 {
		cli = fmt.Sprintf("%s -u %s:'%s'", cli, has.UserName, has.Password)
	}
	if len(has.token) > 0 && !has.HasHeader(HEADER_AUTHORIZATION) {
		has.SetHeader(HEADER_AUTHORIZATION, fmt.Sprintf("Bearer %s", has.token))
	}
	if !has.HasHeader(HEADER_CONTENT_TYPE) {
		has.SetContentTypeHeaderJSON()
	}
	if len(has.Headers) > 0 {
		for k, v := range has.Headers {
			cli = fmt.Sprintf("%s -H \"%s: %s\"", cli, k, v)
		}
	}
	if strings.Contains(uri, "?") {
		realroute = uri[0:strings.Index(uri, "?")]
		cli = fmt.Sprintf("%s -G%s", cli, routeToDataEncodedQuery(uri[strings.Index(uri, "?")+1:]))
	} else {
		realroute = uri
		if len(has.QueryParams) > 0 {
			cli = fmt.Sprintf("%s -G%s", cli, has.GetQueryParamsAsCurlParams())
		}
	}
	cli = fmt.Sprintf("%s %s", cli, has.getBaseURL())
	if len(realroute) > 0 {
		cli = fmt.Sprintf("%s%s", cli, realroute)
	}
	VerbosePrintf("\tcli=%s", cli)
	VerbosePrintf("END HttpApiStruct::BuildCurlCLI(%s,%s)", method, uri)
	return cli
}

func (has *HttpApiStruct) httpApiCallLocal(method string, uri string) error {
	VerbosePrintf("BEGIN httpApiCallLocal(%s,%s)", method, uri)
	if len(has.Fqdn) == 0 {
		return errors.New("missing ip address for api call")
	}

	for h := range has.Headers {
		if strings.EqualFold(h, HEADER_AUTHORIZATION) {
			fmt.Printf("Authorization header === %s\n", has.Headers[h])
			break
		}
	}

	VerbosePrintf("\t\thttpApiCallLocal:::: uri=%q", uri)
	VerbosePrintf("\t\thttpApiCallLocal:::: FixURI(uri)=%q", FixURI(uri))
	apiURL := has.getBaseURL() + FixURI(uri)
	//HTTP client capable of ignoring self-signed certificate
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if has.Timeout == 0 {
		has.Timeout = httpapi_default_timeout
	}

	if GetDryRun() {
		fmt.Printf("DRY RUN: curl -X %s %s\n", method, apiURL)
		return nil
	}

	client := &http.Client{Transport: tr, Timeout: time.Duration(has.Timeout) * time.Second}
	// Create a new  request
	VerbosePrintln(method + " " + apiURL)
	var req *http.Request
	var err error
	if !has.IsPayloadEmpty() {
		VerbosePrintln("payload is not empty")
		req, err = http.NewRequest(strings.ToUpper(method), apiURL, bytes.NewBuffer(has.requestPayload))
		if err != nil {
			VerbosePrintln(fmt.Sprintf("END (error! 1) httpApiCallLocal(%s,%s)", method, uri))
			return err
		}
	} else {
		VerbosePrintln("payload is empty")
		req, err = http.NewRequest(strings.ToUpper(method), apiURL, nil)
		if err != nil {
			VerbosePrintln(fmt.Sprintf("END (error! 2) httpApiCallLocal(%s,%s)", method, uri))
			return err
		}
	}
	// Add basic authentication header
	// auth := has.UserName + ":" + has.Password
	// authEncoded := base64.StdEncoding.EncodeToString([]byte(auth))
	// req.Header.Add("Authorization", "Basic "+authEncoded)
	// req.Header.Set("Content-Type", "application/json")

	// Add headers
	for k, v := range has.Headers {
		req.Header.Add(k, v)
	}

	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		VerbosePrintln(fmt.Sprintf("END (error! 3) httpApiCallLocal(%s,%s)", method, uri))
		return err
	}
	defer resp.Body.Close()
	has.statusCode = resp.StatusCode

	if has.statusCode == http.StatusBadRequest {
		VerbosePrintln("======================")
		VerbosePrintln("status was bad request")
		VerbosePrintf("\tbut payload was %s", string(has.requestPayload))
		VerbosePrintf("\tand response body was %s", string(has.responseBody))
		VerbosePrintln("======================")
	}

	VerbosePrintf("ignore conflict? %s", TrueIsYes(has.IgnoreConflict))
	if resp.StatusCode == http.StatusConflict && has.IgnoreConflict {
		VerbosePrintln("status was conflict.. and now it's ok, because we're ignoring conflict")
		has.statusCode = http.StatusOK
	} else {
		// Read response body
		has.responseBody, err = io.ReadAll(resp.Body)
		VerbosePrintln("start====")
		VerbosePrintln(string(has.responseBody))
		VerbosePrintln("end====")
		if err != nil {
			VerbosePrintln(fmt.Sprintf("END (error! 4) httpApiCallLocal(%s,%s)", method, uri))
			return err
		}
	}
	if has.statusCode != http.StatusOK && has.statusCode != http.StatusNoContent {
		VerbosePrintln(fmt.Sprintf("END (error! 5) adminApiCallLocal(%s,%s)", method, uri))
		VerbosePrintln(fmt.Sprintf("len of body was %d bytes", len(has.responseBody)))
		return errors.New("status code was " + strconv.Itoa(has.statusCode))
	}
	VerbosePrintf("END httpApiCallLocal(%s,%s)", method, uri)
	return nil
}

func (has *HttpApiStruct) httpApiCallRemote(method string, uri string) error {
	VerbosePrintf("BEGIN httpApiCallRemote(%s,%s)", method, uri)
	cli := has.BuildCurlCLI(method, uri)
	VerbosePrintf("cli=%q", cli)

	if GetDryRun() {
		fmt.Printf("DRY RUN: curl -X %s %s over ssh to %s\n", method, uri, has.ssh.Host)
		return nil
	}

	if has.IsPayloadEmpty() {
		//fmt.Printf("payload is empty??? payload size is %d", len(as.payload))
		if err := has.ssh.SecureRemoteExecution(cli); err != nil {
			has.statusCode = http.StatusServiceUnavailable
			VerbosePrintf("END (err) httpApiCallRemote(%s,%s)", method, uri)
			return err
		}
	} else {
		if err := has.ssh.SecureRemotePipeExecution(has.requestPayload, cli); err != nil {
			VerbosePrintf("\tS R P E returns error: %s", err.Error())
			has.statusCode = http.StatusServiceUnavailable
			VerbosePrintf("END (err) httpApiCallRemote(%s,%s)", method, uri)
			return err
		}
	}
	content := strings.Split(has.ssh.GetBody(), "\n")
	for i := 0; i < len(content); i++ {
		if strings.HasPrefix(content[i], "< HTTP/") {
			//fmt.Println(fmt.Sprintf("whats does 14 look like? %q", content[i][14:]))
			s := strings.Split(content[i][:14], " ")
			var err error
			//			"< HTTP/1.1 400"
			has.statusCode, err = strconv.Atoi(s[len(s)-1])
			if err != nil {
				VerbosePrintln("header parse error")
				VerbosePrintln(fmt.Sprintf("failed to parse this line: %q", content[i]))
				has.statusCode = http.StatusServiceUnavailable
				VerbosePrintf("END (err) adminApiCallRemote(%s,%s)", method, uri)
				return err
			}
			break
		}
	}
	var stopper int
	for i := 0; i < len(content); i++ {
		//alfredo.VerbosePrintln(fmt.Sprintf("**** content[%d] = %s\n", i, content[i]))
		if strings.Contains(content[i], "data not shown") ||
			strings.HasPrefix(content[i], "*") ||
			strings.HasPrefix(content[i], "<") ||
			strings.HasPrefix(content[i], ">") {
			content[i] = ""
		}
	}
	for i := 0; i < len(content); i++ {
		if len(content[i]) > 0 {
			stopper = i
			break
		}
	}
	content = content[stopper:]
	b := strings.Join(content, "")
	has.responseBody = ([]byte)(b)
	VerbosePrintf("\thttpApiCallRemote() -- status code was %d (before correction)", has.statusCode)
	VerbosePrintf("\thttpApiCallRemote() -- len of body was %d (before correction)", len(has.responseBody))
	if len(has.responseBody) > 0 && has.statusCode == http.StatusNoContent {
		has.statusCode = http.StatusOK
	} else if len(has.responseBody) == 0 && has.statusCode == 0 {
		has.statusCode = http.StatusNoContent
	} else if has.statusCode == 0 {
		has.statusCode = http.StatusOK
	} else if has.statusCode > 0 && has.statusCode < 400 && (has.statusCode != http.StatusOK && has.statusCode != http.StatusNoContent) {
		has.statusCode = http.StatusBadRequest
	}
	VerbosePrintf("\thttpApiCallRemote() -- status code was %d (after correction)", has.statusCode)
	if has.statusCode == http.StatusOK || has.statusCode == http.StatusNoContent {
		VerbosePrintln("returning nil because 200-level code found!")
		VerbosePrintf("END httpApiCallRemote(%s,%s)", method, uri)
		return nil
	}
	VerbosePrintf("END (err) httpApiCallRemote(%s,%s)", method, uri)
	return fmt.Errorf("returning non-200 level status %d", has.statusCode)
}

func fixThisPair(p string) string {
	miniList := strings.Split(p, "=")
	return fmt.Sprintf("%s=%s", miniList[0], url.QueryEscape(miniList[1]))
}

func FixURI(u string) string {
	VerbosePrintf("BEGIN FixURI(%s)", u)
	//no query
	if !strings.Contains(u, "?") {
		return u
	}
	urisplits := strings.Split(u, "?")
	if len(urisplits) > 2 {
		panic("URI should not have more than one ?")
	}
	//should have exactly 2; uri on the left and query on the right
	queryList := strings.Split(urisplits[1], "&")
	for i := 0; i < len(queryList); i++ {
		fixedPair := fixThisPair(queryList[i])
		queryList[i] = fixedPair
	}
	VerbosePrintln("END FixURI()")
	return strings.ReplaceAll(fmt.Sprintf("%s?%s", urisplits[0], strings.Join(queryList, "&")), "%40", "@")
}

func (has *HttpApiStruct) adjustHeaders() {
	if !has.HasHeader(HEADER_AUTHORIZATION) {
		if len(has.token) > 0 {
			has.SetHeader(HEADER_AUTHORIZATION, fmt.Sprintf("Bearer %s", has.token))
		} else if len(has.UserName) > 0 && len(has.Password) > 0 {
			auth := fmt.Sprintf("%s:%s", has.UserName, has.Password)
			has.SetHeader(HEADER_AUTHORIZATION, fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(auth))))
		}
	}
	if !has.HasHeader(HEADER_CONTENT_TYPE) {
		has.SetContentTypeHeaderJSON()
	}
}

func (has *HttpApiStruct) HttpApiCall(method string, uri string) error {
	VerbosePrintf("BEGIN HttpApiStruct::HttpApiCall(%s,%s)", method, uri)
	if len(has.Fqdn) == 0 {
		return fmt.Errorf("missing fqdn / ip address")
	}
	has.adjustHeaders()
	if has.GetSSHEnabled() || (has.forceRemote && !has.forceLocal) {
		VerbosePrintf("calling has.HttpApiCallRemote(%s,%s)", method, uri)
		if strings.Contains(uri, "%40") {
			panic("uri contains %40")
		}
		return has.httpApiCallRemote(method, uri)
	}
	VerbosePrintf("END HttpApiStruct::HttpApiCall(%s,%s) - hand over to HttpApiCallLocal", method, uri)
	return has.httpApiCallLocal(method, uri)
}

func (has *HttpApiStruct) HammerTest() error {
	VerbosePrintln("BEGIN HttpApiStruct::HammerTest()")
	if len(has.Fqdn) == 0 {
		return fmt.Errorf("missing ip address / fqdn")
	}

	if len(has.ssh.Host) == 0 {
		return errors.New("ssh is not enabled for this HttpApiStruct")
	}

	VerbosePrintln("END HttpApiStruct::HammerTest() --> handoff to sshstruct::hammertest")
	return has.ssh.HammerTest()
}

type TokenType struct {
	Token string `json:"token"`
}

func (has *HttpApiStruct) AcquireTokenFromPasscode(passcode string) error {
	VerbosePrintln("BEGIN HttpApiStruct::AcquireTokenFromPasscode()")
	if len(has.Fqdn) == 0 {
		return fmt.Errorf("missing ip address / fqdn")
	}

	// if len(has.ssh.Host) == 0 {
	// 	return errors.New("ssh is not enabled for this HttpApiStruct")
	// }

	has.SetPayload([]byte(fmt.Sprintf(`{"passcode":"%s"}`, passcode)))

	if err := has.HttpApiCall("POST", "/login"); err != nil {
		return err
	}

	if has.GetStatusCode() != http.StatusOK {
		return fmt.Errorf("status code was %d", has.GetStatusCode())
	}

	var pct TokenType

	json.Unmarshal(has.GetResponseBody(), &pct)

	has.token = pct.Token

	return nil
}

func (has *HttpApiStruct) ParseFromURL(urlStr string) (string, error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", err
	}

	has.Secure = parsedURL.Scheme == "https"
	has.Fqdn = parsedURL.Hostname()

	port := parsedURL.Port()
	if port == "" {
		if has.Secure {
			has.Port = 443
		} else {
			has.Port = 80
		}
	} else {
		has.Port, err = strconv.Atoi(port)
		if err != nil {
			return "", err
		}
	}

	return parsedURL.RequestURI(), nil
}

func (has *HttpApiStruct) GetToken() string {
	return has.token
}
