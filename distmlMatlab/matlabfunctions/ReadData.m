function [x, y] = ReadData(  xin, yin  )
%READDATA Summary of this function goes here
%   Detailed explanation goes here
    x = importdata(xin);
    y = importdata(yin);
end

