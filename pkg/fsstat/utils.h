#include <string.h>
#include <errno.h>

char* getCError(){
	return strerror(errno);
}