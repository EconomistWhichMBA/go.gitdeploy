/*
This was pulled from Jay Bill: https://gist.github.com/jaybill/2876519
*/
package main

import (
  "os"
  "io"
  "io/ioutil"
)
 
// Copies file source to destination dest.
func CopyFile(source string, dest string) (err error) {
    sf, err := os.Open(source)
    if err != nil {
        return err
    }
    defer sf.Close()
    df, err := os.Create(dest)
    if err != nil {
        return err
    }
    defer df.Close()
    _, err = io.Copy(df, sf)
    if err == nil {
        si, err := os.Stat(source)
        if err != nil {
            err = os.Chmod(dest, si.Mode())
        }
 
    }
 
    return
}
 
// Recursively copies a directory tree, attempting to preserve permissions. 
// Source directory must exist, destination directory must *not* exist. 
func CopyDir(source string, dest string) (err error) {
 
    // get properties of source dir
    fi, err := os.Stat(source)
    if err != nil {
        return err
    }
 
    if !fi.IsDir() {
        return &CopyError{"Source is not a directory"}
    }
 
    // ensure dest dir does not already exist
 
    _, err = os.Open(dest)
    if !os.IsNotExist(err) {
        return &CopyError{"Destination already exists"}
    }
 
    // create dest dir
 
    err = os.MkdirAll(dest, fi.Mode())
    if err != nil {
        return err
    }
 
    entries, err := ioutil.ReadDir(source)
 
    for _, entry := range entries {
 
        sfp := source + "/" + entry.Name()
        dfp := dest + "/" + entry.Name()
        if entry.IsDir() {
            err = CopyDir(sfp, dfp)
            if err != nil {
                log(err.Error())
            }
        } else {
            // perform copy         
            err = CopyFile(sfp, dfp)
            if err != nil {
                log(err.Error())
            }
        }
 
    }
    return
}
 
// A struct for returning custom error messages
type CopyError struct {
    What string
}
 
// Returns the error message defined in What as a string
func (e *CopyError) Error() string {
    return e.What
}
