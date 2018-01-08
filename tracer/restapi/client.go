package restapi

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/levigross/grequests"
	"github.com/pkg/errors"
)

var (
	ErrConflict = errors.New("conflict")
)

// Client helps calling the Goapptrace REST API client.
type Client struct {
	BaseUrl string
	s       *grequests.Session
}

// Init initialize the Goapptrace REST API client.
func (c *Client) Init() error {
	c.s = grequests.NewSession(nil)
	return nil
}

// url construct an absolute URL from a relative URL.
func (c Client) url(relativeUrls ...string) string {
	return c.BaseUrl + "/api/v0.1" + strings.Join(relativeUrls, "/")
}

// Servers returns Log server list.
func (c Client) Servers() ([]ServerStatus, error) {
	var res Servers

	r, err := c.s.Get(c.url("/servers"), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to GET /servers")
	}
	if !r.Ok {
		return nil, errors.Errorf("GET /servers returned unexpected status code: %d", r.StatusCode)
	}

	err = r.JSON(&res)
	if err != nil {
		return nil, errors.Wrap(err, "GET /servers returned invalid JSON")
	}
	return res.Servers, nil
}

// Logs returns a list of log status.
func (c Client) Logs() ([]LogStatus, error) {
	var res Logs

	r, err := c.s.Get(c.url("/logs"), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to GET /logs")
	}
	if r.StatusCode != http.StatusOK {
		return nil, errors.Errorf("GET /logs returned unexpected status code. expected 200, but %d", r.StatusCode)
	}

	err = r.JSON(&res)
	if err != nil {
		return nil, errors.Wrap(err, "GET /logs returned invalid JSON")
	}
	return res.Logs, nil
}

// RemoveLog removes the specified log
func (c Client) RemoveLog(id string) error {
	url := c.url("/log", id)
	res, err := c.s.Delete(url, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to DELETE %s", url)
	}
	if res.StatusCode != http.StatusNoContent {
		return errors.Errorf("DELETE %s returned unexpected status code. expected 200, but %d", url, res.StatusCode)
	}
	return nil
}

// LogStatus returns latest log status
func (c Client) LogStatus(id string) (res LogStatus, err error) {
	var r *grequests.Response
	url := c.url("/log", id)
	r, err = c.s.Get(url, nil)
	if err != nil {
		err = errors.Wrapf(err, "failed to GET %s", url)
		return
	}
	if r.StatusCode != http.StatusOK {
		err = errors.Wrapf(err, "GET %s returned unexpected status code. expected 200, but %d", url, r.StatusCode)
		return
	}

	err = r.JSON(&res)
	if err != nil {
		err = errors.Wrapf(err, "GET %s returned invalid JSON", url)
		return
	}
	return
}

// UpdateLogStatus updates the log status.
// If update operation conflicts, it returns ErrConflict.
func (c Client) UpdateLogStatus(id string, status LogStatus) (newStatus LogStatus, err error) {
	var r *grequests.Response
	url := c.url("/log", id)

	r, err = c.s.Put(url, &grequests.RequestOptions{
		Params: map[string]string{
			"version": strconv.Itoa(status.Version),
		},
	})
	if err != nil {
		err = errors.Wrapf(err, "failed to PUT %s", url)
		return
	}
	switch r.StatusCode {
	case http.StatusOK:
		err = r.JSON(&newStatus)
		return
	case http.StatusConflict:
		err = ErrConflict
		return
	default:
		err = errors.Wrapf(err, "PUT %s returned unexpected status code. expected 200 or 409, but %d", url, r.StatusCode)
		return
	}
}

// SearchFuncCalls filters the function call log records.
func (c Client) SearchFuncCalls(id string, so SearchFuncCallParams) (chan FuncCall, error) {
	url := c.url("/log", id, "func-call", "search")

	r, err := c.s.Get(url, &grequests.RequestOptions{
		Params: so.ToParamMap(),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to GET %s", url)
	}
	if r.StatusCode != http.StatusOK {
		return nil, errors.Wrapf(err, "GET %s returned unexpected status code. expected 200, but %d", url, r.StatusCode)
	}

	dec := json.NewDecoder(r)
	ch := make(chan FuncCall, 1<<20)
	go func() {
		defer close(ch)
		for {
			var fc FuncCall
			if err := dec.Decode(&fc); err != nil {
				log.Println(err)
				return
			}
			ch <- fc
		}
	}()
	return ch, nil
}
func (c Client) Func(logID, funcID string) (f FuncInfo, err error) {
	var r *grequests.Response
	url := c.url("/log", logID, "symbol", "func", funcID)

	r, err = c.s.Get(url, nil)
	if err != nil {
		err = errors.Wrapf(err, "failed to GET %s", url)
		return
	}
	if r.StatusCode != http.StatusOK {
		err = errors.Wrapf(err, "GET %s returned unexpected status code. expected 200, but %d", url, r.StatusCode)
		return
	}

	err = r.JSON(&f)
	return
}
func (c Client) FuncStatus(logID, funcStatusID string) (f FuncStatusInfo, err error) {
	var r *grequests.Response
	url := c.url("/log", logID, "symbol", "func-status", funcStatusID)

	r, err = c.s.Get(url, nil)
	if err != nil {
		err = errors.Wrapf(err, "failed to GET %s", url)
		return
	}
	if r.StatusCode != http.StatusOK {
		err = errors.Wrapf(err, "GET %s returned unexpected status code. expected 200, but %d", url, r.StatusCode)
		return
	}

	err = r.JSON(&f)
	return
}
