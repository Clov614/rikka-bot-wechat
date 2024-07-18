// @Author Clover
// @Data 2024/7/5 下午6:11:00
// @Desc
package serializer

import (
	"fmt"
	"testing"
)

func TestSave(t *testing.T) {
	args := struct {
		Name string
		Age  int
		Desc string
		desc string
	}{
		"陈sir",
		19,
		"很帅",
		"test",
	}

	err := Save("", "", &args)
	if err != nil {
		t.Error(err)
	}

	if err = Save("./test", "01", &args); err != nil {
		t.Error(err)
	}

	if err = Save("./test/testdir", "02", args); err != nil {
		t.Error(err)
	}

	type testType string
	var ts = []testType{"测试值", "测试值01", "测试值02", "\"测试值03\""}
	if err = Save("", "", ts); err != nil {
		t.Error(err)
	}
}

func TestNormalTypeNil(t *testing.T) {
	type testType string
	var ts = []testType{"测试值", "测试值01", "测试值02", "\"测试值03\""}
	if err := Save("", "", ts); err != nil {
		t.Error(err)
	}
}

func TestLoad(t *testing.T) {
	args := struct {
		Name string
		Age  int
		Desc string
		desc string
	}{
		"陈sir",
		19,
		"很帅",
		"test",
	}
	if err := Save("./test", "01", &args); err != nil {
		t.Error(err)
	}

	// Load
	var args2 struct {
		Name string
		Age  int
		Desc string
		desc string
	}
	if err := Load("./test", "01", &args2); err != nil {
		t.Error(err)
	}
	fmt.Println(args2)
	// Output:
	// {陈sir 19 很帅 }

}

func TestSavePtr(t *testing.T) {
	type StringType string

	type testType struct {
		Name *StringType
		Age  int
	}

	var s = StringType("hello")

	testObj := testType{
		Name: &s,
		Age:  19,
	}

	err := Save("./test/cacheMsg", fmt.Sprintf("rikkaMsg%d", 1), testObj)
	if err != nil {
		t.Error(err)
	}
	t.Logf("%#v", testObj)

}
