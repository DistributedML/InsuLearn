function [ weight, datasize ] = PredictModel( x, y, modelv )
    model = AddModel(modelv);
    yh = predict(model,x);
    weight = nnz(yh ~=y)./length(yh);
    datasize = length(y);
    classifier2DplotDT(x,y,model);
end

