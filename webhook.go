package alfredo

import (
	"bytes"
	"fmt"
	"net/http"
)

type WebHookStruct struct {
	WebHookURL string
}

func (wh WebHookStruct) SendMsg(msg string) error {
	// Send a POST request to the webhook URL with the payload
	resp, err := http.Post(wh.WebHookURL, "application/json", bytes.NewBuffer([]byte(fmt.Sprintf(`{"text": "%s"}`, msg))))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	// Check the response status
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", resp.StatusCode)
	}
	return nil
}
