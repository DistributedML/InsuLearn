function [] = multiclassifier2DplotDT(X,y,dmodel)
y(y==-1) = 2;

increment = 100;

clf;
plot(X(y==1,1),X(y==1,2),'g+');hold on;
plot(X(y==2,1),X(y==2,2),'bo');

domain1 = xlim;
domain1 = domain1(1):(domain1(2)-domain1(1))/increment:domain1(2);
domain2 = ylim;
domain2 = domain2(1):(domain2(2)-domain2(1))/increment:domain2(2);

d1 = repmat(domain1',[1 length(domain1)]);
d2 = repmat(domain2,[length(domain2) 1]);
% heat = zeros(size(domain1,2),size(domain2,2),3);
zData = zeros(size(domain1,2),size(domain2,2));
n = length(dmodel.a);
for i=1:n
    if dmodel.a(i) ~= 0
        vals = predict(dmodel.models{i},[d1(:) d2(:)]);
        vals(vals==-1) = 2;
        if size(vals,1) ~= length(d1(:))
            error('Output of model.predict should have T rows');
        elseif size(vals,2) ~= 1
            error('Output of model.predict should have 1 column');
        end
        temp = reshape(vals,size(d1));
        zData = zData + (temp==1).*dmodel.a(i)./n;
    end
end
zData = zData./max(zData(:));

contourf(d1,d2,zData,'EdgeColor','none');
c = linspace(0,0.8,n*2)';
z = zeros(size(c));
% cm1 = cat(1,cat(2,z,c,z),cat(2,z,z,c(end:-1:1)));
cm2 = cat(2,z,c,c(end:-1:1));
colormap(cm2)

plot(X(y==1,1),X(y==1,2),'g+');hold on;
plot(X(y==2,1),X(y==2,2),'bo');
contourf(d1,d2,(zData<0.5),0,'r','Fill','off');