import random

class NodeOperator:
    def __init__(self,k,strategy,network_status):
        self.strategy = strategy
        self.network_status = network_status
        self.k3=k

    #open_bonds: bonds adopted by DE
    def acquire(self,open_bonds):
        if self.strategy=="total": #total strategy
            self.k3 = 1
        else: 
            self.k3 = random.triangular(0,1,self.network_status) # determine new k3 based network status
        return open_bonds * self.k3