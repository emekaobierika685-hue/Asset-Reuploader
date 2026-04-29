package ide

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/emekaobierika685-hue/Asset-Reuploader/internal/roblox"
)

var UploadSoundErrors = struct {
	ErrNotLoggedIn       error
	ErrTokenInvalid      error
	ErrInappropriateName error
}{
	ErrNotLoggedIn:       errors.New("not logged in"),
	ErrTokenInvalid:      errors.New("XSRF token validation failed"),
	ErrInappropriateName: errors.New("inappropriate name or description"),
}

type uploadSoundRequest struct {
	Name              string  `json:"name"`
	File              string  `json:"file"`
	GroupID           int64   `json:"groupId,omitempty"`
	EstimatedFileSize int64   `json:"estimatedFileSize"`
	EstimatedDuration float64 `json:"estimatedDuration"`
	AssetPrivacy      int32   `json:"assetPrivacy"`
}

type uploadSoundResponse struct {
	ID     int64  `json:"id,omitempty"`
	Name   string `json:"name,omitempty"`
	Errors []struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"errors,omitempty"`
}

func newUploadSoundRequest(name string, data *bytes.Buffer, groupID int64) (*http.Request, error) {
	var buffer bytes.Buffer
	encoder := base64.NewEncoder(base64.StdEncoding, &buffer)
	size := int64(data.Len())
	if _, err := io.Copy(encoder, data); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}

	body := uploadSoundRequest{
		Name:              name,
		File:              buffer.String(),
		EstimatedFileSize: size,
		GroupID:           groupID,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", "https://publish.roblox.com/v1/audio", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "RobloxStudio/WinInet")
	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func NewUploadSoundHandler(
	c *roblox.Client,
	name,
	description string,
	data *bytes.Buffer,
	groupID ...int64,
) (func() (int64, error), error) {
	group := int64(0)
	if len(groupID) > 0 {
		group = groupID[0]
	}

	req, err := newUploadSoundRequest(name, data, group)
	if err != nil {
		return func() (int64, error) { return 0, nil }, err
	}

	return func() (int64, error) {
		req.AddCookie(&http.Cookie{
			Name:  ".ROBLOSECURITY",
			Value: c.Cookie,
		})
		req.Header.Set("x-csrf-token", c.GetToken())

		resp, err := c.DoRequest(req)
		if err != nil {
			return 0, err
		}
		defer resp.Body.Close()

		var response uploadSoundResponse
		json.NewDecoder(resp.Body).Decode(&response)

		switch resp.StatusCode {
		case http.StatusOK:
			return response.ID, nil
		case http.StatusBadRequest:
			if response.Errors == nil {
				return 0, errors.New(resp.Status)
			}
			message := response.Errors[0].Message
			if message == "Audio name or description is moderated." {
				req, _ = newUploadSoundRequest("[Censored]", data, group)
				return 0, UploadSoundErrors.ErrInappropriateName
			}
			return 0, errors.New(message)
		case http.StatusUnauthorized:
			if response.Errors == nil {
				return 0, errors.New(resp.Status)
			}
			return 0, UploadSoundErrors.ErrNotLoggedIn
		case http.StatusForbidden:
			c.SetToken(resp.Header.Get("x-csrf-token"))
			return 0, UploadSoundErrors.ErrTokenInvalid
		default:
			if response.Errors != nil {
				return 0, errors.New(response.Errors[0].Message)
			}
			return 0, errors.New(resp.Status)
		}
	}, nil
}
