package restapi

import (
	"encoding/json"
	"io"
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
	url := c.url("/servers")
	err := c.getJSON(url, nil, &res)
	if err != nil {
		return nil, err
	}
	return res.Servers, nil
}

// Logs returns a list of log status.
func (c Client) Logs() ([]LogStatus, error) {
	var res Logs
	url := c.url("/logs")
	err := c.getJSON(url, nil, &res)
	if err != nil {
		return nil, err
	}
	return res.Logs, nil
}

// RemoveLog removes the specified log
func (c Client) RemoveLog(id string) error {
	url := c.url("/log", id)
	return c.delete(url, nil)
}

// LogStatus returns latest log status
func (c Client) LogStatus(id string) (res LogStatus, err error) {
	url := c.url("/log", id)
	err = c.getJSON(url, nil, res)
	return
}

// UpdateLogStatus updates the log status.
// If update operation conflicts, it returns ErrConflict.
func (c Client) UpdateLogStatus(id string, status LogStatus) (newStatus LogStatus, err error) {
	url := c.url("/log", id)
	ro := &grequests.RequestOptions{
		Params: map[string]string{
			"version": strconv.Itoa(status.Version),
		},
	}
	err = c.putJSON(url, ro, &newStatus)
	return
}

// SearchFuncCalls filters the function call log records.
func (c Client) SearchFuncCalls(id string, so SearchFuncCallParams) (chan FuncCall, error) {
	url := c.url("/log", id, "func-call", "search")
	ro := &grequests.RequestOptions{
		Params: so.ToParamMap(),
	}
	r, err := c.get(url, ro)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(r)
	ch := make(chan FuncCall, 1<<20)
	go func() {
		defer r.Close() // nolint: errcheck
		defer close(ch)
		for {
			var fc FuncCall
			if err := dec.Decode(&fc); err != nil {
				if err == io.EOF {
					return
				}
				log.Println(err)
				return
			}
			ch <- fc
		}
	}()
	return ch, nil
}
func (c Client) Func(logID, funcID string) (f FuncInfo, err error) {
	url := c.url("/log", logID, "symbol", "func", funcID)
	err = c.getJSON(url, nil, &f)
	return
}
func (c Client) FuncStatus(logID, funcStatusID string) (f FuncStatusInfo, err error) {
	url := c.url("/log", logID, "symbol", "func-status", funcStatusID)
	err = c.getJSON(url, nil, &f)
	return
}

func (c Client) Goroutines(logID string) (gl chan Goroutine, err error) {
	var r *grequests.Response
	url := c.url("/log", logID, "symbol", "goroutines", "search")
	r, err = c.get(url, nil)
	if err != nil {
		return
	}

	dec := json.NewDecoder(r)
	ch := make(chan Goroutine, 1<<20)
	go func() {
		defer r.Close() // nolint: errcheck
		defer close(ch)
		for {
			var g Goroutine
			if err := dec.Decode(&g); err != nil {
				if err == io.EOF {
					return
				}
				log.Println(err)
				return
			}
			ch <- g
		}
	}()
	return ch, nil
}

func (c Client) get(url string, ro *grequests.RequestOptions) (*grequests.Response, error) {
	r, err := wrapResp(c.s.Get(url, nil))
	if err != nil {
		return nil, err
	}
	switch r.StatusCode {
	case http.StatusOK:
		return r, nil
	default:
		return nil, errUnexpStatus(r, []int{
			http.StatusOK,
		})
	}
}
func (c Client) getJSON(url string, ro *grequests.RequestOptions, data interface{}) (err error) {
	var r *grequests.Response
	r, err = c.get(url, nil)
	if err != nil {
		return
	}
	defer r.Close() // nolint: errcheck
	err = errors.Wrapf(r.JSON(&data), "GET %s returned invalid JSON", url)
	return
}
func (c Client) delete(url string, ro *grequests.RequestOptions) error {
	r, err := wrapResp(c.s.Delete(url, ro))
	if err != nil {
		return err
	}
	switch r.StatusCode {
	case http.StatusNoContent:
		return r.Close()
	default:
		defer r.Close() // nolint: errcheck
		return errUnexpStatus(r, []int{
			http.StatusNoContent,
		})
	}
}
func (c Client) put(url string, ro *grequests.RequestOptions) (*grequests.Response, error) {
	r, err := wrapResp(c.s.Put(url, ro))
	if err != nil {
		return nil, err
	}

	switch r.StatusCode {
	case http.StatusOK:
		return r, nil
	case http.StatusConflict:
		r.Close() // nolint: errcheck
		return nil, ErrConflict
	default:
		r.Close() // nolint: errcheck
		return nil, errors.Wrapf(err, "PUT %s returned unexpected status code. expected 200 or 409, but %d", url, r.StatusCode)
	}
}
func (c Client) putJSON(url string, ro *grequests.RequestOptions, data interface{}) error {
	r, err := c.put(url, nil)
	if err != nil {
		return err
	}
	defer r.Close() // nolint: errcheck
	return errors.Wrapf(r.JSON(&data), "PUT %s returned invalid JSON", url)
}
