package main

import (
	"fmt"
	"reflect"
	"time"
)

type User struct {
	Id        int64
	Username  string
	Password  string
	Logintime time.Time
}



func Struct2Map(obj interface{}) map[string]interface{} {
	t := reflect.TypeOf(obj) // 获取 obj 的类型信息
	v := reflect.ValueOf(obj)

	if t.Kind() == reflect.Ptr { // 如果是指针，则获取其所指向的元素
		t = t.Elem()
		v = v.Elem()
	}

	var data = make(map[string]interface{})
	if t.Kind() == reflect.Struct { // 只有结构体可以获取其字段信息
		for i := 0; i < t.NumField(); i++ {
			data[t.Field(i).Name] = v.Field(i).Interface()
		}

	}
	return data
}

func main() {
	user := User{5, "zhangsan", "pwd", time.Now()}
	data := Struct2Map(user)
	fmt.Println(data)

	mapToStruct(data, &user)
}

func mapToStruct(m interface{}, pointer interface{}) {
	elem, ok := pointer.(reflect.Value)
	fmt.Println(ok)
	fmt.Println(elem.Kind())

	if !ok {
		rv := reflect.ValueOf(pointer)
		e := reflect.New(rv.Type().Elem()).Elem()
		fmt.Println(e.Addr())
		fmt.Println(rv.IsNil())

		if kind := rv.Kind(); kind != reflect.Ptr {
			fmt.Printf("object pointer should be type of '*struct', but got '%v' \n", kind)
		}
		// Using IsNil on reflect.Ptr variable is OK.
		if !rv.IsValid() || rv.IsNil() {
			fmt.Printf("object pointer cannot be nil")
			return
		}
		elem = rv.Elem()
	}

	fmt.Println(elem.Type())

	if elem.Kind() == reflect.Ptr {
		if !elem.IsValid() || elem.IsNil() {
			e := reflect.New(elem.Type().Elem()).Elem()
			elem.Set(e.Addr())
			elem = e
		} else {
			// Assign value with interface Set.
			// Note that only pointer can implement interface Set.
			//if v, ok := elem.Interface().(apiUnmarshalValue); ok {
			//	v.UnmarshalValue(params)
			//	return nil
			//}
		}
	}

}