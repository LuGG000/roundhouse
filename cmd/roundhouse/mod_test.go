package main

import (
	"errors"
	"reflect"
	"testing"
)

// recorder returns a run function that records every call and fails on the n-th (1-based; 0 never).
func recorder(failAt int) (func(args ...string) error, *[][]string) {
	var calls [][]string
	return func(args ...string) error {
		calls = append(calls, args)
		if len(calls) == failAt {
			return errors.New("boom")
		}
		return nil
	}, &calls
}

func TestInstallModAddsTheMarketplaceThenInstalls(t *testing.T) {
	run, calls := recorder(0)
	if err := installMod(run, "LuGG000/roundhouse", "user"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{
		{"plugin", "marketplace", "add", "LuGG000/roundhouse", "--scope", "user"},
		{"plugin", "install", "roundhouse-mod@roundhouse", "--scope", "user"},
	}
	if !reflect.DeepEqual(*calls, want) {
		t.Errorf("calls = %v, want %v", *calls, want)
	}
}

func TestInstallModUsesTheGivenSourceAndScope(t *testing.T) {
	run, calls := recorder(0)
	if err := installMod(run, `C:\src\roundhouse`, "project"); err != nil {
		t.Fatal(err)
	}
	if got := (*calls)[0]; got[3] != `C:\src\roundhouse` || got[5] != "project" {
		t.Errorf("marketplace call = %v", got)
	}
}

func TestInstallModStopsWhenTheMarketplaceFails(t *testing.T) {
	run, calls := recorder(1)
	err := installMod(run, "x/y", "user")
	if err == nil || len(*calls) != 1 {
		t.Errorf("err = %v after %d calls; want an error and no install attempt", err, len(*calls))
	}
}

func TestInstallModReportsAFailingInstall(t *testing.T) {
	run, _ := recorder(2)
	if err := installMod(run, "x/y", "user"); err == nil {
		t.Error("want an error when the install fails")
	}
}

func TestUninstallMod(t *testing.T) {
	run, calls := recorder(0)
	if err := uninstallMod(run, "user"); err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"plugin", "uninstall", "roundhouse-mod@roundhouse", "--scope", "user"}}
	if !reflect.DeepEqual(*calls, want) {
		t.Errorf("calls = %v, want %v", *calls, want)
	}

	failing, _ := recorder(1)
	if err := uninstallMod(failing, "user"); err == nil {
		t.Error("want an error when the uninstall fails")
	}
}
