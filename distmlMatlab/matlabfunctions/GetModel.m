function [ modelv, c ] = GetModel( model )
%GETMODEL Summary of this function goes here
%   Detailed explanation goes here
    charset = char(['a':'z' '0':'9']);
    R = charset(ceil(36*rand(1,20)));
    save(R,'model');
    fid = fopen([R '.mat']);
    [modelv, c] = fread(fid,'ubit16');
    fclose(fid);
    delete([R '.mat']);
end

