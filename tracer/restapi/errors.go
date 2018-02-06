package restapi

import (
	"fmt"
	"strconv"

	"github.com/levigross/grequests"
	"github.com/pkg/errors"
)

// wrapResp wraps error object by errFailedResp().
func wrapResp(r *grequests.Response, err error) (*grequests.Response, error) {
	if err != nil {
		return nil, errFailedResp(r, err)
	}
	return r, nil
}

// errFailedResp wraps error object in the user friendly error message.
func errFailedResp(res *grequests.Response, err error) error {
	method := res.RawResponse.Request.Method
	url := res.RawResponse.Request.URL
	return errors.Wrapf(err, "failed to %s %s", method, url)
}

// errUnexpStatus returns a error object.
func errUnexpStatus(res *grequests.Response, expected []int) error {
	method := res.RawResponse.Request.Method
	url := res.RawResponse.Request.URL
	actual := res.StatusCode

	exp := ""
	for i := range expected {
		if i != 0 {
			exp += "|"
		}
		exp += strconv.FormatInt(int64(expected[i]), 10)
	}

	return fmt.Errorf("%s %s returned unexpected status code. expected %s, but %d",
		method, url,
		exp, actual)
}
