function [ gmodel ] = AggregateGlobalModel( gmodel, R, S )
%GETMODEL Summary of this function goes here
%   Detailed explanation goes here

for i=1:length(gmodel.models)
    if ~isempty(gmodel.models{i})
        [~,I] = sort(R,1);
        D = zeros(size(I));
        for j=1:length(gmodel.models)
            D(sub2ind(size(D),I(j,:),1:length(gmodel.models))) = 2^(-(j));
        end
        D(R==0) = 0;
        Dt = sum(D,2);
        Dt = Dt/max(Dt);
        Dt(Dt<median(Dt)) = 0;
        gmodel.a(i) = S(i)*Dt(i);
    end
end

gmodel.a = gmodel.a./sum(gmodel.a);

end

