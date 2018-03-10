package runtime

import "unsafe"

type _GAT_SilentLog struct {
	m    mutex
	logs []interface{}
}

//go:nosplit
func (l *_GAT_SilentLog) Print(args ...interface{}) {
	if l == nil {
		return
	}
	lock(&l.m)
	l.logs = append(l.logs, args...)
	unlock(&l.m)
}

//go:nosplit
func (l *_GAT_SilentLog) Println(args ...interface{}) {
	if l == nil {
		return
	}
	lock(&l.m)
	l.logs = append(l.logs, args...)
	l.logs = append(l.logs, "\n")
	unlock(&l.m)
}

//go:nosplit
func (l *_GAT_SilentLog) Clear() {
	if l == nil {
		return
	}
	lock(&l.m)
	l.logs = l.logs[:0]
	unlock(&l.m)
}

//go:nosplit
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

//go:nosplit
func (l *_GAT_SilentLog) Fatal(s string) {
	l.Show()
	throw(s)
}

// 全てのシンボルを列挙する
func init() {
	for _, env := range environ() {
		const key = "go_debug="

		if len(env) < len(key) {
			continue
		}
		if env[:len(key)] == key {
			// matched
			iterateSymbols(
				func(pc uintptr, name string) {
					println(pc, "func", name)
				},
				func(pc uintptr, file string, line int32) {
					println(pc, file, ":", line)
				},
			)
			break
		}
	}
}

func iterateSymbols(
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

		// ftab is lookup table for function by program counter.
		// moduledataverify1 skips last item of ftab. what is last item???
		nftab := len(datap.ftab) - 1
		for fidx, ft := range datap.ftab[:nftab] {
			log.Println("functab: entry=", ft.entry, " funcoff=", ft.funcoff)

			if ft.entry < datap.minpc && datap.maxpc < ft.entry {
				log.Println()
				log.Println("invalid functab")
				log.Println("\t", datap.minpc, " <= ", ft.entry, " <= ", datap.maxpc)
				log.Println()
				log.Fatal("invalid functab")
				*(*int)(nil) = 0 // not reached
			}

			fp := unsafe.Pointer(&datap.pclntable[ft.funcoff])
			//fn := (*Func)(fp)
			rawfn := (*_func)(fp)
			fi := funcInfo{rawfn, datap}
			if !fi.valid() {
				log.Fatal("runtime.iterateSymbols: invalid funcInfo")
				*(*int)(nil) = 0 // not reached
			}
			log.Println("_func: ", unsafe.Pointer(rawfn))
			log.Println("\tentry=", rawfn.entry)
			log.Println("\tnameoff=", rawfn.nameoff)
			log.Println("\targs=", rawfn.args)
			log.Println("\tpcsp=", rawfn.pcsp)
			log.Println("\tpcfile=", rawfn.pcfile)
			log.Println("\tpcln=", rawfn.pcln)
			log.Println("\tnpcdata=", rawfn.npcdata)
			log.Println("\tnfuncdata=", rawfn.nfuncdata)

			pclns, lines := pcvalueIterate(log, fi, rawfn.pcln, rawfn.entry, strict)
			if len(pclns) == 0 || len(lines) == 0 {
				// invalid
				log.Println()
				log.Println("runtime.iterateSymbols: invalid state")
				log.Println("midx=", midx, " len(modules)=", len(modules), )
				log.Println("fidx=", fidx, " len(ftab)=", len(datap.ftab))
				log.Println()
				log.Fatal("runtime.iterateSymbols: invalid state")
				*(*int)(nil) = 0 // not reached
			}
			if len(pclns) != len(lines) {
				log.Println()
				log.Println("runtime.iterateSymbols: mismatch length: len(pclns)=", len(pclns), " len(lines)=", len(lines))
				log.Println()
				log.Fatal("mismatch length")
				*(*int)(nil) = 0 // not reached
			}

			funcname := funcname(fi)
			log.Println(hex(rawfn.entry), " func: name=", funcname)
			addFunc(rawfn.entry, funcname)

			for i := range pclns {
				pc := pclns[i]
				line := lines[i]

				funcobj := fi._Func()
				file, line2 := funcobj.FileLine(pc)

				if int(line) != line2 {
					log.Println("assert: file=", file)
					log.Println("assert: line =", line)
					log.Println("        line2=", line2)
					log.Println("assert: funcname=", funcname)
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
						log.Println("assert: funcname=", funcname)
						log.Println("  funcobj.Name()=", funcobj.Name())

						if file != f {
							log.Fatal("mismatch file name")
						}
						if int(lines[i]) != l {
							log.Fatal("mismatch line number")
						}
						if funcname != funcobj.Name() {
							log.Fatal("mismatch func name")
						}
					}
				}
			}
		}
	}
}

//go:nosplit
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
