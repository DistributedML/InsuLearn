function [out, c] = FitModel( x, y )
%     model = compact(RegressionTree.fit(x,y));
    model = compact(ClassificationTree.fit(x,y));
    [out, c] = GetModel( model );
end