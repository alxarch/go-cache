# go-cache

Cache structs for Go


## Basic cache

Basic implements `Interface` that stores an unlimited number of items.
It has no eviction policy removal of expired items is the responsibility of the
user (via `Basic#Trim(time.Time)`).

  
