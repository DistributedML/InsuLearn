function [ acc, cor ] = PredictGlobalModel( x, y, modelv )
%PREDICTGLOBALMODEL Summary of this function goes here
%   Detailed explanation goes here
gmodel = AddModel(modelv);
yh = zeros(size(y));

for i=1:length(gmodel.models)
    if ~isempty(gmodel.models{i})
        yh = yh + predict(gmodel.models{i},x)*gmodel.a(i);
    end
end
yh = sign(yh);

acc = nnz(yh==y)/length(yh);
cor = 0;

% Visualization
figure;
multiclassifier2DplotDT(x,y,gmodel);

end

