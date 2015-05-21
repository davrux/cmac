// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// CMAC message authentication code, defined in
// NIST Special Publication SP 800-38B.

package cmac

import (
	"crypto/cipher"
	"errors"
	"hash"
)

const (
	// minimal irreducible polynomial of degree b
	r64  = 0x1b
	r128 = 0x87
)

type cmac struct {
	k1, k2, ci, digest []byte
	p                  int // position in ci
	c                  cipher.Block
}

// NewCMAC returns a new instance of a CMAC message authentication code
// digest using the given cipher.Block.
func NewCMAC(c cipher.Block) (hash.Hash, error) {
	var r byte
	n := c.BlockSize()
	switch n {
	case 64 / 8:
		r = r64
	case 128 / 8:
		r = r128
	default:
		return nil, errors.New("NewCMAC: invalid cipher block size")
	}

	d := new(cmac)
	d.c = c
	d.k1 = make([]byte, n)
	d.k2 = make([]byte, n)
	d.ci = make([]byte, n)
	d.digest = make([]byte, n)

	// Subkey generation, p. 7
	c.Encrypt(d.k1, d.k1)
	if shift1(d.k1, d.k1) != 0 {
		d.k1[n-1] ^= r
	}
	if shift1(d.k1, d.k2) != 0 {
		d.k2[n-1] ^= r
	}

	return d, nil
}

// Reset clears the digest state, starting a new digest.
func (d *cmac) Reset() {
	for i := range d.ci {
		d.ci[i] = 0
	}
	d.p = 0
}

// Write adds the given data to the digest state.
func (d *cmac) Write(p []byte) (n int, err error) {
	// Xor input into ci.
	for _, c := range p {
		// If ci is full, encrypt and start over.
		if d.p >= len(d.ci) {
			d.c.Encrypt(d.ci, d.ci)
			d.p = 0
		}
		d.ci[d.p] ^= c
		d.p++
	}
	return len(p), nil
}

// Sum returns the CMAC digest, one cipher block in length,
// of the data written with Write.
func (d *cmac) Sum(in []byte) []byte {
	// Finish last block, mix in key, encrypt.
	// Don't edit ci, in case caller wants
	// to keep digesting after call to Sum.
	k := d.k1
	if d.p < len(d.digest) {
		k = d.k2
	}
	for i := 0; i < len(d.ci); i++ {
		d.digest[i] = d.ci[i] ^ k[i]
	}
	if d.p < len(d.digest) {
		d.digest[d.p] ^= 0x80
	}
	d.c.Encrypt(d.digest, d.digest)
	return append(in, d.digest...)
}

func (d *cmac) Size() int { return len(d.digest) }

func (d *cmac) BlockSize() int { return d.c.BlockSize() }

func shift1(src, dst []byte) byte {
	var b byte
	for i := len(src) - 1; i >= 0; i-- {
		bb := src[i] >> 7
		dst[i] = src[i]<<1 | b
		b = bb
	}
	return b
}
