#include <stdint.h>
#include <inttypes.h>
#include <windows.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>
#include "engine.h"
#include "_cgo_export.h"

#define MATLAB_FUNCTION_PATH "C:/work/src/github.com/4180122/distbayes/distmlMatlab/matlabfunctions"

void ReadData(Engine *ep, char *x, char *y){
    mxArray *yin = NULL, *xin = NULL, *fpath = NULL;

    xin = mxCreateString(x);
    yin = mxCreateString(y);
    fpath = mxCreateString(MATLAB_FUNCTION_PATH);
    engPutVariable(ep, "xin", xin);
    engPutVariable(ep, "yin", yin);
    engPutVariable(ep, "f_path", fpath);
    engEvalString(ep, "addpath(f_path);");
    engEvalString(ep, "cd(f_path)");
    engEvalString(ep, "[x, y] = ReadData(xin,yin);");

 	mxDestroyArray(xin);
 	mxDestroyArray(yin);
 	mxDestroyArray(fpath);
}

double* FitModel(Engine *ep, double *paramlen){
    mxArray *outlen = NULL, *out = NULL;
    double *modeltemp;
    double modelsize;

    engEvalString(ep, "[out, c] = FitModel(x,y);");

    out = engGetVariable(ep, "out");
    outlen = engGetVariable(ep, "c");
    modeltemp = mxGetPr(out);
    paramlen[0] = mxGetScalar(outlen);

 	//mxDestroyArray(out);
 	mxDestroyArray(outlen);

    return modeltemp;
}

void Predict(Engine *ep, double *model, int paramlen, double *weight, double *size){
    mxArray *in = NULL, *weightmat = NULL, *sizemat = NULL;

    in = mxCreateDoubleMatrix(paramlen, 1, mxREAL);
	memcpy((char *) mxGetPr(in), (char *) model, paramlen*sizeof(double));
    engPutVariable(ep, "in", in);

    engEvalString(ep, "[weight, datasize] = PredictModel(x, y, in);");

    weightmat = engGetVariable(ep, "weight");
    sizemat = engGetVariable(ep, "datasize");
    weight[0] = mxGetScalar(weightmat);
    size[0] = mxGetScalar(sizemat);

 	mxDestroyArray(in);
 	mxDestroyArray(weightmat);
 	mxDestroyArray(sizemat);
}

void PushModel(Engine *ep, double *model, int paramlen, double *rvec, int veclen, double modelC, double location) {
    mxArray *in = NULL, *ind = NULL, *row = NULL, *C = NULL;
    in = mxCreateDoubleMatrix(paramlen, 1, mxREAL);
    row = mxCreateDoubleMatrix(1, veclen, mxREAL);
    C = mxCreateDoubleScalar(modelC);
    ind = mxCreateDoubleScalar(location);
	memcpy((char *) mxGetPr(in), (char *) model, paramlen*sizeof(double));
	memcpy((char *) mxGetPr(row), (char *) rvec, veclen*sizeof(double));
    engPutVariable(ep, "in", in);
    engPutVariable(ep, "Rtemp", row);
    engPutVariable(ep, "Ctemp", C);
    engPutVariable(ep, "ind", ind);
    engEvalString(ep, "gmodel.models{ind+1} = AddModel(in);");
    engEvalString(ep, "R(ind+1,:) = Rtemp;");
    engEvalString(ep, "C(ind+1) = Ctemp;");
 	mxDestroyArray(in);
 	mxDestroyArray(row);
 	mxDestroyArray(C);
 	mxDestroyArray(ind);
}

void InitializeGlobal(Engine *ep, double nodes, double modelD) {
    mxArray *N = NULL, *D = NULL, *fpath = NULL;
    N = mxCreateDoubleScalar(nodes);
    D = mxCreateDoubleScalar(modelD);
    fpath = mxCreateString(MATLAB_FUNCTION_PATH);
    engPutVariable(ep, "N", N);
    engPutVariable(ep, "D", D);
    engPutVariable(ep, "f_path", fpath);
    engEvalString(ep, "addpath(f_path);");
    engEvalString(ep, "cd(f_path)");
    engEvalString(ep, "gmodel.models = cell(N,1)");
    engEvalString(ep, "gmodel.a = zeros(N,1)");
    engEvalString(ep, "R = zeros(N,N)");
    engEvalString(ep, "C = zeros(N,1)");
 	mxDestroyArray(N);
 	mxDestroyArray(D);
}

double* GetGlobalModel(Engine *ep, double *paramlen) {
    mxArray *outlen = NULL, *out = NULL;
    double *modeltemp;
    double modelsize;

    engEvalString(ep, "gmodel = AggregateGlobalModel(gmodel, R, C);");
    engEvalString(ep, "[out, c] = GetModel(gmodel);");

    out = engGetVariable(ep, "out");
    outlen = engGetVariable(ep, "c");
    modeltemp = mxGetPr(out);
    paramlen[0] = mxGetScalar(outlen);

 	//mxDestroyArray(out);
 	mxDestroyArray(outlen);

    return modeltemp;
}


void PredictGlobal(Engine *ep, double *model, int paramlen, double *acc, double *cor){
    mxArray *in = NULL, *accmat = NULL, *cormat = NULL;
    char buffer[257];

    in = mxCreateDoubleMatrix(paramlen, 1, mxREAL);
	memcpy((char *) mxGetPr(in), (char *) model, paramlen*sizeof(double));
    engPutVariable(ep, "in", in);

    engEvalString(ep, "[acc, cor] = PredictGlobalModel(x, y, in);");

    accmat = engGetVariable(ep, "acc");
    cormat = engGetVariable(ep, "cor");
    acc[0] = mxGetScalar(accmat);
    cor[0] = mxGetScalar(cormat);

    buffer[256] = '\0';
	engOutputBuffer(ep, buffer, 256);
	engEvalString(ep, "disp(['Accuracy: ' num2str(acc) ', Correlation :' num2str(cor)])");
	MessageBox ((HWND)NULL, (LPSTR)buffer, (LPSTR) "Results", MB_OK);

 	mxDestroyArray(in);
 	mxDestroyArray(accmat);
 	mxDestroyArray(cormat);
}
