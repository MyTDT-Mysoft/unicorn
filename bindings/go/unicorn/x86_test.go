package unicorn

import (
	"testing"
)

var ADDRESS uint64 = 0x1000000

func MakeUc(mode int, code string) (*Uc, error) {
	mu, err := NewUc(ARCH_X86, mode)
	if err != nil {
		return nil, err
	}
	if err := mu.MemMap(ADDRESS, 2*1024*1024); err != nil {
		return nil, err
	}
	if err := mu.MemWrite(ADDRESS, []byte(code)); err != nil {
		return nil, err
	}
	if err := mu.RegWrite(X86_REG_ECX, 0x1234); err != nil {
		return nil, err
	}
	if err := mu.RegWrite(X86_REG_EDX, 0x7890); err != nil {
		return nil, err
	}
	return mu, nil
}

func TestX86(t *testing.T) {
	code := "\x41\x4a"
	mu, err := MakeUc(MODE_32, code)
	if err != nil {
		t.Fatal(err)
	}
	if err := mu.Start(ADDRESS, ADDRESS+uint64(len(code))); err != nil {
		t.Fatal(err)
	}
	ecx, _ := mu.RegRead(X86_REG_ECX)
	edx, _ := mu.RegRead(X86_REG_EDX)
	if ecx != 0x1235 || edx != 0x788f {
		t.Fatal("Bad register values.")
	}
}

func TestX86InvalidRead(t *testing.T) {
	code := "\x8B\x0D\xAA\xAA\xAA\xAA\x41\x4a"
	mu, err := MakeUc(MODE_32, code)
	if err != nil {
		t.Fatal(err)
	}
	err = mu.Start(ADDRESS, ADDRESS+uint64(len(code)))
	if err.(UcError) != ERR_MEM_READ {
		t.Fatal("Expected ERR_MEM_READ")
	}
	ecx, _ := mu.RegRead(X86_REG_ECX)
	edx, _ := mu.RegRead(X86_REG_EDX)
	if ecx != 0x1234 || edx != 0x7890 {
		t.Fatal("Bad register values.")
	}
}

func TestX86InvalidWrite(t *testing.T) {
	code := "\x89\x0D\xAA\xAA\xAA\xAA\x41\x4a"
	mu, err := MakeUc(MODE_32, code)
	if err != nil {
		t.Fatal(err)
	}
	err = mu.Start(ADDRESS, ADDRESS+uint64(len(code)))
	if err.(UcError) != ERR_MEM_WRITE {
		t.Fatal("Expected ERR_MEM_WRITE")
	}
	ecx, _ := mu.RegRead(X86_REG_ECX)
	edx, _ := mu.RegRead(X86_REG_EDX)
	if ecx != 0x1234 || edx != 0x7890 {
		t.Fatal("Bad register values.")
	}
}

func TestX86InOut(t *testing.T) {
	code := "\x41\xE4\x3F\x4a\xE6\x46\x43"
	mu, err := MakeUc(MODE_32, code)
	if err != nil {
		t.Fatal(err)
	}
	var outVal uint64
	var inCalled, outCalled bool
	mu.HookAdd(HOOK_INSN, func(mu *Uc, port, size uint32) uint32 {
		inCalled = true
		switch size {
		case 1:
			return 0xf1
		case 2:
			return 0xf2
		case 4:
			return 0xf4
		default:
			return 0
		}
	}, X86_INS_IN)
	mu.HookAdd(HOOK_INSN, func(uc *Uc, port, size, value uint32) {
		outCalled = true
		var err error
		switch size {
		case 1:
			outVal, err = mu.RegRead(X86_REG_AL)
		case 2:
			outVal, err = mu.RegRead(X86_REG_AX)
		case 4:
			outVal, err = mu.RegRead(X86_REG_EAX)
		}
		if err != nil {
			t.Fatal(err)
		}
	}, X86_INS_OUT)
	if err := mu.Start(ADDRESS, ADDRESS+uint64(len(code))); err != nil {
		t.Fatal(err)
	}
	if !inCalled || !outCalled {
		t.Fatal("Ports not accessed.")
	}
	if outVal != 0xf1 {
		t.Fatal("Incorrect OUT value.")
	}
}

func TestX86Syscall(t *testing.T) {
	code := "\x0f\x05"
	mu, err := MakeUc(MODE_64, code)
	if err != nil {
		t.Fatal(err)
	}
	mu.HookAdd(HOOK_INSN, func(mu *Uc) {
		rax, _ := mu.RegRead(X86_REG_RAX)
		mu.RegWrite(X86_REG_RAX, rax+1)
	}, X86_INS_SYSCALL)
	mu.RegWrite(X86_REG_RAX, 0x100)
	err = mu.Start(ADDRESS, ADDRESS+uint64(len(code)))
	if err != nil {
		t.Fatal(err)
	}
	v, _ := mu.RegRead(X86_REG_RAX)
	if v != 0x101 {
		t.Fatal("Incorrect syscall return value.")
	}
}
