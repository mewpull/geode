is io

link "c/io.c"

include "std:mem"

# Printing bindings
func nomangle _VM2ioN5print(string format, ...) ...
func nomangle _VM2ioN6format(string format, ...) string ...

# func print(string format, ...) {
# 	cprintf(format);
# }

# _VN2io5print
# func __openfile(string path, string mode) byte* ...
# func __readchar(byte* fp) byte ...
# func __fileeof(byte* fp) int ...
# func __filewritestring(byte*fp, string data) int ...
