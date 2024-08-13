import random

class BondHolder:
    def __init__(self,k,strategy):
        self.strategy = strategy
        self.k1=k
    
    def acquire(self,open_bonds, k2):
        if self.strategy == "total": #total strategy
            self.k1 = 1
        else: 
            self.k1 = random.triangular(0,k2,k2) # determine new k1 based on current k2 
        return open_bonds * self.k1