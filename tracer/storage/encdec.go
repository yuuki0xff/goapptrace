package storage

import (
	"encoding/gob"
	"io"
)

// gobエンコードされたデータをFileに書き込む。
type Encoder struct {
	File File

	a   io.WriteCloser // AppendOnly
	enc *gob.Encoder
}

// Fileからgobエンコードされたデータを読み込む。
type Decoder struct {
	File File

	r   io.ReadCloser // ReadOnly
	dec *gob.Decoder
}

func (e *Encoder) Open() (err error) {
	e.a, err = e.File.OpenAppendOnly()
	e.enc = gob.NewEncoder(e.a)
	return
}

func (d *Decoder) Open() (err error) {
	d.r, err = d.File.OpenReadOnly()
	d.dec = gob.NewDecoder(d.r)
	return
}

// ストリームから次の値を読み込み、dataに格納する。
// ストリームの終端に達した場合、io.EOFを返す。
func (d *Decoder) Read(data interface{}) error {
	return d.dec.Decode(data)
}

// Walk()は、次の値をnewPtr()が確保したメモリ領域に読み込み、callback()を呼び出す。
// これを、ストリームの終端に達するか、callbackがエラーを返すまで繰り返し行う。
// newPtr()とcallback()は1つの値を読み込むたびに呼び出される。
func (d *Decoder) Walk(newPtr func() interface{}, callback func(interface{}) error) error {
	for {
		val := newPtr()
		if err := d.Read(val); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err := callback(val); err != nil {
			return err
		}
	}
}

func (e *Encoder) Append(data interface{}) (err error) {
	return e.enc.Encode(data)
}

func (e *Encoder) Close() (err error) {
	if e.a != nil {
		err = e.a.Close()
		e.a = nil
		e.enc = nil
	}
	return
}

func (d *Decoder) Close() (err error) {
	if d.r != nil {
		err = d.r.Close()
		d.r = nil
		d.dec = nil
	}
	return
}
