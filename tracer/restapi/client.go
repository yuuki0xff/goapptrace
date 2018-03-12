package restapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/levigross/grequests"
	"github.com/pkg/errors"
)

const (
	UserAgent = "goapptrace-restapi-client"
)

var (
	ErrConflict = errors.New("conflict")
)

// Client helps calling the Goapptrace REST API client.
type Client struct {
	BaseUrl string
	s       *grequests.Session
}

type ClientWithCtx struct {
	Client
	UseCache bool

	ctx   context.Context
	cache apiCache
}

type apiCache struct {
	m    sync.RWMutex
	logs map[string]*logCache
}

type logCache struct {
	m  sync.RWMutex
	fs map[uintptr]*GoLineInfo
	f  map[uintptr]*FuncInfo
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

// ro returns an initialized RequestOptions struct.
func (c ClientWithCtx) ro() grequests.RequestOptions {
	return grequests.RequestOptions{
		UserAgent: UserAgent,
		Context:   c.ctx,
	}
}

// WithCtx returns a new ClientWithCtx object with specified context.
//
// this method MUST use value receiver.
func (c Client) WithCtx(ctx context.Context) ClientWithCtx {
	var cc ClientWithCtx
	cc.Client = c
	cc.UseCache = true
	cc.ctx = ctx
	cc.cache.init()
	return cc
}

// Servers returns Log server list.
func (c ClientWithCtx) Servers() ([]ServerStatus, error) {
	var res Servers
	url := c.url("/servers")
	ro := c.ro()
	err := c.getJSON(url, &ro, &res)
	if err != nil {
		return nil, err
	}
	return res.Servers, nil
}

// Logs returns a list of log status.
func (c ClientWithCtx) Logs() ([]LogStatus, error) {
	var res Logs
	url := c.url("/logs")
	ro := c.ro()
	err := c.getJSON(url, &ro, &res)
	if err != nil {
		return nil, err
	}
	return res.Logs, nil
}

// RemoveLog removes the specified log
func (c ClientWithCtx) RemoveLog(id string) error {
	url := c.url("/log", id)
	ro := c.ro()
	return c.delete(url, &ro)
}

// LogStatus returns latest log status
func (c ClientWithCtx) LogStatus(id string) (res LogStatus, err error) {
	url := c.url("/log", id)
	ro := c.ro()
	err = c.getJSON(url, &ro, res)
	return
}

// UpdateLogStatus updates the log status.
// If update operation conflicts, it returns ErrConflict.
func (c ClientWithCtx) UpdateLogStatus(id string, status LogStatus) (newStatus LogStatus, err error) {
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
func (c ClientWithCtx) SearchFuncCalls(id string, so SearchFuncCallParams) (chan FuncCall, error) {
	url := c.url("/log", id, "func-call", "search")
	ro := c.ro()
	ro.Params = so.ToParamMap()
	r, err := c.get(url, &ro)
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
func (c ClientWithCtx) GoFunc(logID string, pc uintptr) (f FuncInfo, err error) {
	if c.UseCache {
		fcache := c.cache.Log(logID).Func(pc)
		if fcache != nil {
			// fast path
			f = *fcache
			return
		}
	}

	// slow path
	url := c.url("/log", logID, "symbol", "func", FormatUintptr(pc))
	ro := c.ro()
	err = c.getJSON(url, &ro, &f)

	if err == nil {
		// validation
		if f.Entry == 0 || f.Entry > pc {
			err = fmt.Errorf("validation error: FuncInfo=%+v", f)
			log.Panic(errors.WithStack(err))
		}
	}
	if err == nil && c.UseCache {
		c.cache.AddLog(logID).AddFunc(f)
	}
	return
}
func (c ClientWithCtx) GoLine(logID string, pc uintptr) (fs GoLineInfo, err error) {
	if c.UseCache {
		fscache := c.cache.Log(logID).GoLine(pc)
		if fscache != nil {
			// fast path
			fs = *fscache
			return
		}
	}

	// slow path
	url := c.url("/log", logID, "symbol", "func-status", FormatUintptr(pc))
	ro := c.ro()
	err = c.getJSON(url, &ro, &fs)

	if err == nil {
		// validation
		if fs.PC == 0 || fs.PC > pc {
			err = fmt.Errorf("validation error: GoLineInfo=%+v", fs)
			log.Panic(errors.WithStack(err))
		}
	}
	if err == nil && c.UseCache {
		c.cache.AddLog(logID).AddGoLine(fs)
	}
	return
}

func (c ClientWithCtx) Goroutines(logID string) (gl chan Goroutine, err error) {
	var r *grequests.Response
	url := c.url("/log", logID, "goroutines", "search")
	ro := c.ro()
	r, err = c.get(url, &ro)
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
	r, err := wrapResp(c.s.Get(url, ro))
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
	r, err = c.get(url, ro)
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
	r, err := c.put(url, ro)
	if err != nil {
		return err
	}
	defer r.Close() // nolint: errcheck
	return errors.Wrapf(r.JSON(&data), "PUT %s returned invalid JSON", url)
}

func (c *apiCache) init() {
	c.logs = map[string]*logCache{}
}

func (c *apiCache) Log(logID string) *logCache {
	c.m.RLock()
	l := c.logs[logID]
	c.m.RUnlock()
	return l
}

func (c *apiCache) AddLog(logID string) *logCache {
	c.m.Lock()
	l := c.logs[logID]
	if l == nil {
		l = &logCache{}
		l.init()
		c.logs[logID] = l
	}
	c.m.Unlock()
	return l
}

func (c *logCache) init() {
	c.fs = map[uintptr]*GoLineInfo{}
	c.f = map[uintptr]*FuncInfo{}
}

func (c *logCache) Func(pc uintptr) *FuncInfo {
	if c == nil {
		return nil
	}
	c.m.RLock()
	f := c.f[pc]
	c.m.RUnlock()
	return f
}

func (c *logCache) AddFunc(f FuncInfo) {
	c.m.Lock()
	if _, ok := c.f[f.Entry]; ok {
		fp := &FuncInfo{}
		*fp = f
		c.f[f.Entry] = fp
	}
	c.m.Unlock()
}

func (c *logCache) GoLine(pc uintptr) *GoLineInfo {
	if c == nil {
		return nil
	}
	c.m.RLock()
	fs := c.fs[pc]
	c.m.RUnlock()
	return fs
}

func (c *logCache) AddGoLine(fs GoLineInfo) {
	c.m.Lock()
	if _, ok := c.fs[fs.PC]; ok {
		fsp := &GoLineInfo{}
		*fsp = fs
		c.fs[fs.PC] = fsp
	}
	c.m.Unlock()
}

func FormatUintptr(ptr uintptr) string {
	return strconv.FormatUint(uint64(ptr), 10)
}

func ParseUintptr(s string) (uintptr, error) {
	ptr, err := strconv.ParseUint(s, 10, 64)
	return uintptr(ptr), err
}
