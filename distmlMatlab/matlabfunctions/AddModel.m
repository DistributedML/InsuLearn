function [ model ] = AddModel( modelv )
%ADDMODEL Summary of this function goes here
%   Detailed explanation goes here
if length(modelv) > 1
    charset = char(['a':'z' '0':'9']);
    R = charset(ceil(36*rand(1,20)));
    fid2 = fopen([R '.mat'],'w');
    fwrite(fid2,char(modelv),'ubit16');
    fclose(fid2);
    load(R);
    delete([R '.mat']);
end
end

