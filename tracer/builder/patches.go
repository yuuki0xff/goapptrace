package builder

const runtimePatch = `
package runtime

import "unsafe"

// GoID returns the Goroutine ID.
//go:nosplit
func GoID() int64 {
	gp := getg()
	return gp.goid
}

// IterateSymbols walks the symbols table in this process.
func IterateSymbols(
	addModule func(minpc, maxpc uintptr, name string),
	addFunc func(pc uintptr, name string),
	addLine func(pc uintptr, file string, line int32),
) {
	const strict = true
	const superStrict = true
	var log *_GAT_SilentLog
	// disable debug log output.
	//log = &_GAT_SilentLog{}

	defer func() {
		log.Show()
	}()

	modules := activeModules()
	for midx, datap := range modules {
		log.Println("datap: minpc=", datap.minpc, " maxpc=", datap.maxpc)
		addModule(datap.minpc, datap.maxpc, datap.modulename)

		// ftab is lookup table for function by program counter.
		// moduledataverify1 skips last item of ftab. what is last item???
		nftab := len(datap.ftab) - 1
		for fidx := 0; fidx < nftab; fidx++ {
			ft := datap.ftab[fidx]
			log.Println("functab: entry=", ft.entry, " funcoff=", ft.funcoff)

			if ft.entry < datap.minpc && datap.maxpc < ft.entry {
				log.Println()
				log.Println("invalid functab")
				log.Println("\t", datap.minpc, " <= ", ft.entry, " <= ", datap.maxpc)
				log.Println()
				log.Fatal("invalid functab")
			}

			fp := unsafe.Pointer(&datap.pclntable[ft.funcoff])
			//fn := (*Func)(fp)
			rawfn := (*_func)(fp)
			fi := funcInfo{rawfn, datap}
			if !fi.valid() {
				log.Fatal("runtime.iterateSymbols: invalid funcInfo")
			}
			log.Println("_func: ", unsafe.Pointer(rawfn))
			log.Println("\tentry=", rawfn.entry)
			log.Println("\tnameoff=", rawfn.nameoff, " name=", funcname(fi))
			log.Println("\targs=", rawfn.args)
			log.Println("\tpcsp=", rawfn.pcsp)
			log.Println("\tpcfile=", rawfn.pcfile)
			log.Println("\tpcln=", rawfn.pcln)
			log.Println("\tnpcdata=", rawfn.npcdata)
			log.Println("\tnfuncdata=", rawfn.nfuncdata)

			if *rawfn == (_func{}) {
				log.Fatal("invalid functab")
			} else if rawfn.entry != 0 && rawfn.nameoff != 0 && rawfn.args == 0 && rawfn.pcsp == 0 && rawfn.pcfile == 0 && rawfn.pcln == 0 && rawfn.npcdata == 0 && rawfn.nfuncdata == 0 {
				fname := funcname(fi)
				if len(fname) >= 7 && fname[len(fname)-7:] == "(.text)" {
					// TODO: What is this functab??  What is this for??
					// next functab seems the cgo function. Also, this and next functab seem to have the same entry point.

					// validation
					if fidx+1 >= nftab {
						// this is last function in this module.
						log.Fatal("missing next functab")
					}
					if ft.entry != datap.ftab[fidx+1].entry {
						log.Println("current ftab.entry=", ft.entry)
						log.Println("   next ftab.entry=", datap.ftab[fidx+1].entry)
						log.Fatal("mismatch entry point")
					}
				} else if len(fname) >= 5 && fname[:5] == "_cgo_" {
					// cgo function
				} else {
					// it is probably the built-in function
				}

				continue
			}

			pclns, lines := pcvalueIterate(log, fi, rawfn.pcln, rawfn.entry, strict)
			if len(pclns) == 0 || len(lines) == 0 {
				// invalid
				log.Println()
				log.Println("runtime.iterateSymbols: invalid state")
				log.Println("\tmidx=", midx, " len(modules)=", len(modules), )
				log.Println("\tfidx=", fidx, " len(ftab)=", len(datap.ftab))
				log.Println("\tlen(pclns)=", len(pclns), " len(lines)=", len(lines))
				log.Println()
				log.Fatal("runtime.iterateSymbols: invalid state")
			}
			if len(pclns) != len(lines) {
				log.Println()
				log.Println("runtime.iterateSymbols: mismatch length: len(pclns)=", len(pclns), " len(lines)=", len(lines))
				log.Println()
				log.Fatal("mismatch length")
			}

			fname := funcname(fi)
			log.Println(hex(rawfn.entry), " func: name=", fname)
			addFunc(rawfn.entry, fname)

			for i := range pclns {
				pc := pclns[i]
				line := lines[i]

				funcobj := fi._Func()
				file, line2 := funcobj.FileLine(pc)

				if int(line) != line2 {
					log.Println("assert: file=", file)
					log.Println("assert: line =", line)
					log.Println("        line2=", line2)
					log.Println("assert: funcname=", fname)
					log.Println("  funcobj.Name()=", funcobj.Name())
					log.Fatal("mismatch line number")
				}

				log.Println(hex(pclns[i]), " file=", file, " line=", line)
				addLine(pc, file, line)

				if superStrict {
					log.Println("assert: pc=", pc)
					//funcobj := FuncForPC(pclns[i])
					funcobj := FuncForPC(fi.entry)
					if funcobj != nil {
						log.Println("assert: funcobj=", unsafe.Pointer(funcobj))
						f, l := funcobj.FileLine(pc)
						log.Println("assert: file=", file)
						log.Println("        f   =", f, " (untrusted infomation)")
						log.Println("assert: line=", line)
						log.Println("        l   =", l)
						log.Println("assert: funcname=", fname)
						log.Println("  funcobj.Name()=", funcobj.Name())

						if file != f {
							log.Fatal("mismatch file name")
						}
						if int(lines[i]) != l {
							log.Fatal("mismatch line number")
						}
						if fname != funcobj.Name() {
							log.Fatal("mismatch func name")
						}
					}
				}
			}
		}
	}
}

func pcvalueIterate(log *_GAT_SilentLog, f funcInfo, off int32, targetpc uintptr, strict bool) (pcs []uintptr, vals []int32) {
	if off == 0 {
		return
	}

	if !f.valid() {
		if strict && panicking == 0 {
			log.Println("runtime: no module data for ", hex(f.entry))
			log.Fatal("no module data")
		}
		return
	}
	datap := f.datap
	p := datap.pclntable[off:]
	pc := f.entry
	val := int32(-1)
	for {
		oldpc := pc
		var ok bool
		p, ok = step(p, &pc, &val, pc == f.entry)
		if !ok {
			return
		}
		pcs = append(pcs, oldpc)
		vals = append(vals, val)
	}

	// If there was a table, it should have covered all program counters.
	// If not, something is wrong.
	if panicking != 0 || !strict {
		return
	}

	log.Println("runtime: invalid pc-encoded table f=", funcname(f), " pc=", hex(pc), " targetpc=", hex(targetpc), " tab=", p)
	return
}

type _GAT_SilentLog struct {
	m    mutex
	logs []interface{}
}

func (l *_GAT_SilentLog) Print(args ...interface{}) {
	if l == nil {
		return
	}
	lock(&l.m)
	l.logs = append(l.logs, args...)
	unlock(&l.m)
}

func (l *_GAT_SilentLog) Println(args ...interface{}) {
	if l == nil {
		return
	}
	lock(&l.m)
	l.logs = append(l.logs, args...)
	l.logs = append(l.logs, "\n")
	unlock(&l.m)
}

func (l *_GAT_SilentLog) Clear() {
	if l == nil {
		return
	}
	lock(&l.m)
	l.logs = l.logs[:0]
	unlock(&l.m)
}

func (l *_GAT_SilentLog) Show() {
	if l == nil {
		return
	}
	lock(&l.m)
	for _, ival := range l.logs {
		switch val := ival.(type) {
		case hex:
			printhex(uint64(val))
		case unsafe.Pointer:
			print(val)
		default:
			printany(val)
		}
	}
	unlock(&l.m)
}

func (l *_GAT_SilentLog) Fatal(s string) {
	if l != nil {
		l.Show()
	}
	throw(s)
	*(*int)(nil) = 0 // not reached
	return
}
`
