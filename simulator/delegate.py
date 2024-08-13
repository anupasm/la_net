import random

class Delegate:
    def __init__(self,k,strategy):
        self.strategy = strategy
        self.k2=k
        pass
    
    #open_bonds: bonds adopted by BH
    #k3:ratio adopted by NO in the last rebase
    def acquire(self,open_bonds, k3):
        if self.strategy=="total": #total strategy
            self.k2 = 1
        else: 
            self.k2 = random.triangular(0,k3,k3) # determine new k2 based on current k3
        return open_bonds * self.k2