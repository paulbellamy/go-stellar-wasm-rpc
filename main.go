// An experiment in running the stellar wasmi contract runtime from go.
// Luckily, the wasm runtime compiles to wasm itself! So we can use a wasm
// runtime to call the rust package from
// github.com/paulbellamy/rs-stellar-wasm-browser, and use that to run our
// actual contract wasm.
package main

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io/ioutil"

	wasmer "github.com/wasmerio/wasmer-go/wasmer"
)

const PIXEL_NFT_WASM = "AGFzbQEAAAABDgNgAAF+YAF/AGABfwF+AwUEAAABAgUDAQARBhkDfwFBgIDAAAt/AEG8gcAAC38AQcCBwAALBzUFBm1lbW9yeQIABXBpeGVsAAAFb3duZXIAAQpfX2RhdGFfZW5kAwELX19oZWFwX2Jhc2UDAgqYAwQJAELxv+u+lQEL+QICA38CfgJ+IwBBEGsiACQAA0ACQAJAIAACfyACQQpGBEAgAEEIaiAEQgSGQgmENwMAQQAMAQsgAkEKRwRAQgEhAyACQYCAQGstAAAiAUHfAEYNAiABrSEDAkACQCABQTBrQf8BcUEKTwRAIAFBwQBrQf8BcUEaSQ0BIAFB4QBrQf8BcUEaSQ0CIABBATYCBCAAQQhqIAE2AgBBAQwECyADQi59IQMMBAsgA0I1fSEDDAMLIANCO30hAwwCCyAAQQA2AgQgAEEIakEKNgIAQQELNgIADAELIAJBAWohAiADIARCBoaEIQQMAQsLIAAoAgBFBEAgACkDCCAAQRBqJAAMAQsjAEEgayIAJAAgAEEUakEANgIAIABBrIHAADYCECAAQgE3AgQgAEEONgIcIABBjIHAADYCGCAAIABBGGo2AgAjAEEgayIBJAAgAUEBOgAYIAFBnIHAADYCFCABIAA2AhAgAUGsgcAANgIMIAFBrIHAADYCCAALCwMAAQsNAELX1o7QiJnYj7V/CwvDAQEAQYCAwAALuQFHQktMTVFWTkNSL1VzZXJzL3BhdWxiZWxsYW15Ly5jYXJnby9naXQvY2hlY2tvdXRzL3JzLXN0ZWxsYXItY29udHJhY3QtZW52LWE3NDU5OGJlZmVmNTk3OGQvNmIzNmZkNS9zdGVsbGFyLWNvbnRyYWN0LWVudi1jb21tb24vc3JjL3N5bWJvbC5yc2V4cGxpY2l0IHBhbmljAAAKABAAggAAAFoAAAAXAAAAAQAAAAAAAAABAAAAAg=="

func main() {
	// load and compile the wasm runtime
	wasmBytes, _ := ioutil.ReadFile("../rs-stellar-wasm-browser/pkg/stellar_wasm_browser_bg.wasm")
	engine := wasmer.NewEngine()
	store := wasmer.NewStore(engine)
	module, _ := wasmer.NewModule(store, wasmBytes)

	// Instantiates the runtime
	importObject := wasmer.NewImportObject()
	instance, err := wasmer.NewInstance(module, importObject)
	if err != nil {
		panic(err)
	}

	// What are we using the runtime for? This is a test call.
	contractIdBase64 := "0000000000000000000000000000000000000000000000000000000000000000"
	wasmBase64 := PIXEL_NFT_WASM
	function := "pixel"
	argsXdrBase64 := ""

	// Get handles to stuff in the wasm
	memory, err := instance.Exports.GetMemory("memory")
	if err != nil {
		panic(err)
	}
	malloc, err := instance.Exports.GetFunction("__wbindgen_malloc")
	if err != nil {
		panic(err)
	}
	realloc, err := instance.Exports.GetFunction("__wbindgen_realloc")
	if err != nil {
		panic(err)
	}
	addToStackPointer, err := instance.Exports.GetFunction("__wbindgen_add_to_stack_pointer")
	if err != nil {
		panic(err)
	}

	// Set up a return pointer for the returned vector.
	// Because we return a vector of bytes, it's returned as ptr and length
	// int32s, so make 16 bytes of space.
	retptrI, err := addToStackPointer(-16)
	if err != nil {
		panic(err)
	}
	retptr := retptrI.(int32)
	defer addToStackPointer(16)

	// Push all our string args into the wasm memory
	ptr0, len0, err := passStringToWasm0(memory, contractIdBase64, malloc, realloc)
	if err != nil {
		panic(err)
	}
	ptr1, len1, err := passStringToWasm0(memory, wasmBase64, malloc, realloc)
	if err != nil {
		panic(err)
	}
	ptr2, len2, err := passStringToWasm0(memory, function, malloc, realloc)
	if err != nil {
		panic(err)
	}
	ptr3, len3, err := passStringToWasm0(memory, argsXdrBase64, malloc, realloc)
	if err != nil {
		panic(err)
	}

	// Gets the exported function from the runtime.
	invokeContract, err := instance.Exports.GetFunction("invoke_contract")
	if err != nil {
		panic(err)
	}

	// call the wasm runtime in wasm
	_, err = invokeContract(
		retptrI,
		ptr0, len0,
		ptr1, len1,
		ptr2, len2,
		ptr3, len3,
	)
	if err != nil {
		panic(err)
	}

	// debugging
	// fmt.Println("retptr:", retptr, "expected: 1048560")
	// fmt.Println("retptr/4+0:", retptr/4+0, "expected: 262140")
	// fmt.Println("retptr/4+1:", retptr/4+1, "expected: 262141")
	// fmt.Println("ptr,len[0]:", ptr0, len0, "expected: 1114120 64")
	// fmt.Println("ptr,len[1]:", ptr1, len1, "expected: 1114192 972")
	// fmt.Println("ptr,len[2]:", ptr2, len2, "expected: 1115168 5")
	// fmt.Println("ptr,len[3]:", ptr3, len3, "expected: 4 0")
	// fmt.Println("result:", result)

	// Get the returned vector of bytes. The result is 2 int32s (8 bytes). First
	// is pointer to the vector, second is the vector length.
	mem := memory.Data()
	// Need to read 4 bytes into an int32 for each of these.
	r := bytes.NewReader(mem[retptr:])
	var r0 int32 // returned vector pointer
	if err := binary.Read(r, binary.LittleEndian, &r0); err != nil {
		panic(err)
	}
	var r1 int32 // returned vector length
	if err := binary.Read(r, binary.LittleEndian, &r1); err != nil {
		panic(err)
	}
	// fmt.Println("r0:", r0, "expected: 1125704")
	// fmt.Println("r1:", r1, "expected: 8")

	// Go get the real array and copy it out from the wasm runtime memory.
	// Probably we could avoid doing the copy here, but without it the slice we
	// return points to the underlying wasm memory, so we could corrupt the state
	// if we changed the result.
	v4, err := getArrayU8FromWasm0(memory, r0, r1)
	if err != nil {
		panic(err)
	}
	// fmt.Println("v4:", v4, "expected: [0 0 0 1 149 125 173 255]")

	// Cleanup.
	free, err := instance.Exports.GetFunction("__wbindgen_free")
	if err != nil {
		panic(err)
	}
	free(r0, r1*1)

	// Print the result as base64-encoded xdr.
	fmt.Println(base64.StdEncoding.EncodeToString(v4)) // 42!
}

func passStringToWasm0(memory *wasmer.Memory, arg string, malloc, _realloc wasmer.NativeFunction) (int32, int, error) {
	ptr, err := malloc(len(arg))
	if err != nil {
		return 0, 0, err
	}
	p := ptr.(int32)
	mem := memory.Data()
	pp := mem[p : int(p)+len(arg)]
	n := copy(pp[:], arg)
	return p, n, nil
}

func getArrayU8FromWasm0(memory *wasmer.Memory, ptr, length int32) ([]byte, error) {
	dst := make([]byte, length)
	copy(dst, memory.Data()[ptr:ptr+length])
	return dst, nil
}
