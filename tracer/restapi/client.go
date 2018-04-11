package restapi

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/levigross/grequests"
	"github.com/pkg/errors"
	"github.com/yuuki0xff/goapptrace/tracer/types"
	"golang.org/x/sync/errgroup"
)

const (
	UserAgent = "goapptrace-restapi-client"
)

var (
	msgUseCache = "<cache>"

	ErrConflict         = errors.New("conflict")
	ErrNotFoundGoModule = errors.New("not found GoModule")
	ErrNotFoundGoFunc   = errors.New("not found GoFunc")
	ErrNotFoundGoLine   = errors.New("not found GoLine")
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
	// types.Symbols へのポインタ
	s unsafe.Pointer
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
		Params:    map[string]string{},
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
func (c *ClientWithCtx) SyncSymbols(logID string) error {
	var data types.SymbolsData
	url := c.url("/log", logID, "symbols")
	ro := c.ro()
	err := c.getJSON(url, &ro, &data)
	if err != nil {
		return err
	}

	s := &types.Symbols{}
	s.Load(data)
	c.cache.AddLog(logID).SetSymbols(s)
	return nil
}

func (c *ClientWithCtx) Symbols(logID string) (s *types.Symbols, err error) {
	s = c.cache.Log(logID).Symbols()
	if s != nil {
		return
	}
	err = c.SyncSymbols(logID)
	if err != nil {
		return
	}
	s = c.cache.Log(logID).Symbols()
	return
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
func (c ClientWithCtx) Logs() ([]types.LogInfo, error) {
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

// LogInfo returns latest log status
func (c ClientWithCtx) LogInfo(id string) (res types.LogInfo, err error) {
	url := c.url("/log", id)
	ro := c.ro()
	err = c.getJSON(url, &ro, &res)
	return
}

// SetLogInfo updates the log status.
// If update operation conflicts, it returns ErrConflict.
func (c ClientWithCtx) SetLogInfo(id string, new types.LogInfo) (updated types.LogInfo, err error) {
	url := c.url("/log", id)
	ro := c.ro()
	ro.JSON = new
	err = c.putJSON(url, &ro, &updated)
	return
}

// SearchRaw execute a SQL query and returns result by CSV format.
func (c *ClientWithCtx) SearchRaw(id string, query string) (io.ReadCloser, error) {
	url := c.url("/log", id, "search.csv")
	ro := c.ro()
	ro.Params["sql"] = query
	return c.get(url, &ro)
}

// Search executes a SQL statement.
func (c *ClientWithCtx) Search(id string, query string) (chan<- []string, *errgroup.Group) {
	ch := make(chan []string, 1024)
	eg := &errgroup.Group{}

	eg.Go(func() error {
		defer close(ch)
		rr, err := c.SearchRaw(id, query)
		if err != nil {
			return err
		}
		defer rr.Close() // nolint

		r := csv.NewReader(rr)
		for {
			rec, err := r.Read()
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			select {
			case ch <- rec:
			case <-c.ctx.Done():
				return c.ctx.Err()
			}
		}
	})
	return ch, eg
}

// SearchFuncLogs filters the function call log records.
func (c ClientWithCtx) SearchFuncLogs(id string, so SearchFuncLogParams) (<-chan types.FuncLog, *errgroup.Group) {
	ch := make(chan types.FuncLog, 1024)
	eg := &errgroup.Group{}

	eg.Go(func() error {
		defer close(ch)
		url := c.url("/log", id, "func-call", "search")
		ro := c.ro()
		ro.Params = so.ToParamMap()
		r, err := c.get(url, &ro)
		if err != nil {
			return err
		}
		defer r.Close() // nolint: errcheck

		dec := json.NewDecoder(r)
		for {
			var fc types.FuncLog
			if err := dec.Decode(&fc); err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			ch <- fc
		}
	})
	return ch, eg
}
func (c ClientWithCtx) GoModule(logID string, pc uintptr) (m types.GoModule, err error) {
	if c.UseCache {
		// fast path
		var s *types.Symbols
		var ok bool
		s, err = c.Symbols(logID)
		if err != nil {
			return
		}
		m, ok = s.GoModule(pc)
		if !ok {
			err = ErrNotFoundGoModule
			return
		}
		c.validateGoModule(pc, msgUseCache, m)
		return
	}

	// slow path
	url := c.url("/log", logID, "symbol", "module", FormatUintptr(pc))
	ro := c.ro()
	err = c.getJSON(url, &ro, &m)

	if err == nil {
		c.validateGoModule(pc, url, m)
	}
	return
}
func (c ClientWithCtx) GoFunc(logID string, pc uintptr) (f types.GoFunc, err error) {
	if c.UseCache {
		// fast path
		var s *types.Symbols
		var ok bool
		s, err = c.Symbols(logID)
		if err != nil {
			return
		}
		f, ok = s.GoFunc(pc)
		if !ok {
			err = ErrNotFoundGoFunc
			return
		}
		c.validateGoFunc(pc, msgUseCache, f)
		return
	}

	// slow path
	url := c.url("/log", logID, "symbol", "func", FormatUintptr(pc))
	ro := c.ro()
	err = c.getJSON(url, &ro, &f)

	if err == nil {
		c.validateGoFunc(pc, url, f)
	}
	return
}
func (c ClientWithCtx) GoLine(logID string, pc uintptr) (l types.GoLine, err error) {
	if c.UseCache {
		// fast path
		var s *types.Symbols
		var ok bool
		s, err = c.Symbols(logID)
		if err != nil {
			return
		}
		l, ok = s.GoLine(pc)
		if !ok {
			err = ErrNotFoundGoLine
			return
		}
		c.validateGoLine(pc, msgUseCache, l)
		return
	}

	// slow path
	url := c.url("/log", logID, "symbol", "line", FormatUintptr(pc))
	ro := c.ro()
	err = c.getJSON(url, &ro, &l)

	if err == nil {
		c.validateGoLine(pc, url, l)
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

func (c ClientWithCtx) UpdateTraceTargets(tracerID string, names []string) error {
	var targets types.TraceTarget
	targets.Funcs = names

	url := c.url("/tracer", tracerID, "targets")
	ro := c.ro()
	ro.JSON = targets
	r, err := c.put(url, &ro)
	if err != nil {
		return err
	}
	return r.Close()
}

func (c ClientWithCtx) StartTrace(tracerID string, name string) error {
	url := c.url("/tracer", tracerID, "target", "func", name)
	ro := c.ro()
	r, err := c.put(url, &ro)
	if err != nil {
		return err
	}
	return r.Close()
}

func (c ClientWithCtx) StopTrace(tracerID string, name string) error {
	url := c.url("/tracer", tracerID, "target", "func", name)
	ro := c.ro()
	return c.delete(url, &ro)
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
		return nil, errUnexpStatus(r, []int{
			http.StatusOK,
		})
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

func (c *logCache) init() {}

func (c *logCache) SetSymbols(s *types.Symbols) {
	atomic.StorePointer(&c.s, unsafe.Pointer(s)) // nolint: gas
}

func (c *logCache) Symbols() *types.Symbols {
	if c == nil {
		return nil
	}
	sp := atomic.LoadPointer(&c.s)
	return (*types.Symbols)(sp)
}

func (c *logCache) GoModule(pc uintptr) (m types.GoModule, ok bool) {
	s := c.Symbols()
	if s == nil {
		return
	}
	m, ok = s.GoModule(pc)
	return
}

func (c *logCache) GoFunc(pc uintptr) (f types.GoFunc, ok bool) {
	s := c.Symbols()
	if s == nil {
		return
	}
	f, ok = s.GoFunc(pc)
	return
}

func (c *logCache) GoLine(pc uintptr) (l types.GoLine, ok bool) {
	s := c.Symbols()
	if s == nil {
		return
	}
	l, ok = s.GoLine(pc)
	return
}

func FormatUintptr(ptr uintptr) string {
	return strconv.FormatUint(uint64(ptr), 10)
}

func ParseUintptr(s string) (uintptr, error) {
	ptr, err := strconv.ParseUint(s, 10, 64)
	return uintptr(ptr), err
}
