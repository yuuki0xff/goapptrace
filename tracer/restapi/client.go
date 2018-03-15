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
	"github.com/yuuki0xff/goapptrace/tracer/types"
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
	fs map[uintptr]*types.GoLine
	f  map[uintptr]*types.GoFunc
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

func (c *ClientWithCtx) Ctx() context.Context {
	return c.ctx
}

// SyncSymbolsはサーバからシンボルテーブルをダウンロードし、キャッシュする。
// キャッシュを作っておくことで、クライアント側のみでシンボル解決が出来るようになり、高速化が出来る。
func (c *ClientWithCtx) SyncSymbols() error {
	// TODO: not implements
	return nil
}

func (c *ClientWithCtx) Symbols() (*types.Symbols, error) {
	// TODO:
	return nil, nil
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

// SearchFuncLogs filters the function call log records.
func (c ClientWithCtx) SearchFuncLogs(id string, so SearchFuncLogParams) (chan types.FuncLog, error) {
	url := c.url("/log", id, "func-call", "search")
	ro := c.ro()
	ro.Params = so.ToParamMap()
	r, err := c.get(url, &ro)
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(r)
	ch := make(chan types.FuncLog, 1<<20)
	go func() {
		defer r.Close() // nolint: errcheck
		defer close(ch)
		for {
			var fc types.FuncLog
			if err := dec.Decode(&fc); err != nil {
				if err == io.EOF {
					return
				}
				// TODO: ここで発生したエラーを、クライアント側に通知する
				log.Println(err)
				return
			}
			ch <- fc
		}
	}()
	return ch, nil
}
func (c ClientWithCtx) GoModule(logID string, pc uintptr) (m types.GoModule, err error) {
	// TODO: lookup cache
	url := c.url("/log", logID, "symbol", "module", FormatUintptr(pc))
	ro := c.ro()
	err = c.getJSON(url, &ro, &m)

	if err == nil {
		c.validateGoModule(pc, url, m)
	}
	// TODO: add to cache
	return
}
func (c ClientWithCtx) GoFunc(logID string, pc uintptr) (f types.GoFunc, err error) {
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
		c.validateGoFunc(pc, url, f)
	}
	if err == nil && c.UseCache {
		c.cache.AddLog(logID).AddFunc(f)
	}
	return
}
func (c ClientWithCtx) GoLine(logID string, pc uintptr) (l types.GoLine, err error) {
	if c.UseCache {
		fscache := c.cache.Log(logID).GoLine(pc)
		if fscache != nil {
			// fast path
			l = *fscache
			return
		}
	}

	// slow path
	url := c.url("/log", logID, "symbol", "line", FormatUintptr(pc))
	ro := c.ro()
	err = c.getJSON(url, &ro, &l)

	if err == nil {
		c.validateGoLine(pc, url, l)
	}
	if err == nil && c.UseCache {
		c.cache.AddLog(logID).AddGoLine(l)
	}
	return
}

func (c ClientWithCtx) Goroutines(logID string) (gl chan types.Goroutine, err error) {
	var r *grequests.Response
	url := c.url("/log", logID, "goroutines", "search")
	ro := c.ro()
	r, err = c.get(url, &ro)
	if err != nil {
		return
	}

	dec := json.NewDecoder(r)
	ch := make(chan types.Goroutine, 1<<20)
	go func() {
		defer r.Close() // nolint: errcheck
		defer close(ch)
		for {
			var g types.Goroutine
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
func (c Client) validateGoModule(pc uintptr, url string, m types.GoModule) {
	if m.Name == "" || m.MinPC == 0 || m.MaxPC == 0 || pc < m.MinPC || m.MaxPC < pc {
		err := fmt.Errorf("validation error: Module=%+v url=%s", m, url)
		log.Panic(errors.WithStack(err))
	}
}
func (c Client) validateGoFunc(pc uintptr, url string, f types.GoFunc) {
	if f.Entry == 0 || f.Entry > pc {
		err := fmt.Errorf("validation error: GoFunc=%+v url=%s", f, url)
		log.Panic(errors.WithStack(err))
	}
}
func (c Client) validateGoLine(pc uintptr, url string, l types.GoLine) {
	if l.PC == 0 || l.PC > pc {
		err := fmt.Errorf("validation error: GoLine=%+v url=%s", l, url)
		log.Panic(errors.WithStack(err))
	}
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
	c.fs = map[uintptr]*types.GoLine{}
	c.f = map[uintptr]*types.GoFunc{}
}

func (c *logCache) Func(pc uintptr) *types.GoFunc {
	if c == nil {
		return nil
	}
	c.m.RLock()
	f := c.f[pc]
	c.m.RUnlock()
	return f
}

func (c *logCache) AddFunc(f types.GoFunc) {
	c.m.Lock()
	if _, ok := c.f[f.Entry]; ok {
		fp := &types.GoFunc{}
		*fp = f
		c.f[f.Entry] = fp
	}
	c.m.Unlock()
}

func (c *logCache) GoLine(pc uintptr) *types.GoLine {
	if c == nil {
		return nil
	}
	c.m.RLock()
	fs := c.fs[pc]
	c.m.RUnlock()
	return fs
}

func (c *logCache) AddGoLine(fs types.GoLine) {
	c.m.Lock()
	if _, ok := c.fs[fs.PC]; ok {
		fsp := &types.GoLine{}
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
