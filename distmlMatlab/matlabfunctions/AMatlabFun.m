function out = AMatlabFun( a )
    if strcmp(a,'hi')
        out = 'AMatlabFun()';
    else
        out = 'oops!';
    end
    figure; imagesc(rand(100,100));
end