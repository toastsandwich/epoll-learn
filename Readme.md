# Introduction
- epoll monitors a group of file descriptors to see whether I/O is possible in any of them.
- epoll uses red-black tree data structure to keep track of all file descriptors that are being monitored.

# APIs
```c
// creates and epoll object and returns its file descriptor.
// flags allows epoll behavior to be modified. only one valid value EPOLL_CLOEXEC
int epoll_create1(int flags); // gives more control over epoll object
int epoll_create(int size); // deprecated
   ```

```c
// Controls/configures which fds are watched by epoll obj, and for which events.
// events can be ADD, MODIFY, DELETE.
// EPOLL_CTL_ADD: add a fd to the interest list of epoll object
// EPOLL_CTL_MOD: modify already existing fd in epoll object
// EPOLL_CTL_DEL: removes target fd from interest list of epoll object
int epoll_ctl(int epfd, int op, int fd, struct epoll_event* event);
```

```c
// waits for any of events registered for with epoll_ctl, until atleast one occurs or the timeout elapses. Returns occured events, upto maxevents at once. 
// events in an array of struct epoll_event
// timeout set to 0 never blocks, set to -1 blocks forever
int epoll_wait(int epfd, struct epoll_event *events, int maxevents, int timeout);
```


# More on struct epoll_event

```c
struct epoll_event {
	/*
	this field is used to specify which events thee fd should be monitored for
	EPOLLIN: used for read operations
	EPOLLOUT: used for write operations
	EPOLLLET: requests edge-triggered event notification for the fd
	*/
	uint32_t      events; /* Epoll events */
	
	// specify data that kernel should save and return when the fd is ready	
	epoll_data_t  data;  /* User data variable */
}

union epoll_data {
	  void      *ptr;   // ptr to some user-define data, useful when custom obj is associated with a event
    int       fd;     // fd we are interested in
    int32_t   u32;    // stores flags or timeout
    uint64_t  u64;    // stores large values or timeout
};
      
typedef union epoll_data epoll_data_t;
```

# Triggering modes
Two types of triggering modes **edge-triggered** and **level-triggered**.
1. Edge-triggered: call to `epoll_wait` is returned only when a new event is enqueued with the `epoll` object. 
   
2. Level-triggered: call to `epoll_wait` will return immediately



