is color

include "math"

class RGB {
	float r;
	float g;
	float b;
}

func new_rgb(float r, float g, float b) RGB {
	RGB col;
	col.r = r;
	col.g = g;
	col.b = b;	
	return col;
}


func hsv_to_rgb(float h, float s, float v) RGB {
	h = h % 360.0;
	if s = 0 {
		return new_rgb(v, v, v);
	}

	i = (h * 6);
	float C = v * s;
	float X = C * (1 - math:fabs((h / 60.0) % 2));
	float m = v - C;

	RGB col;
	if   0 <= h && h < 60  {col = new_rgb(C, X, 0);}
	if  60 <= h && h < 120 {col = new_rgb(X, C, 0);}
	if 120 <= h && h < 180 {col = new_rgb(0, C, X);}
	if 180 <= h && h < 240 {col = new_rgb(0, X, C);}
	if 240 <= h && h < 300 {col = new_rgb(X, 0, C);}
	if 300 <= h && h < 360 {col = new_rgb(C, 0, X);}

	col.r = (col.r + m) * 255;
	col.g = (col.g + m) * 255;
	col.b = (col.b + m) * 255;
	return col;
}


func equal(RGB a, RGB b) bool {
	return a.r = b.r && a.g = b.g && a.b = c.b;
}