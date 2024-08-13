from static_data import *
from bondholder import BondHolder
from delegate import Delegate
from nodeoperator import NodeOperator
from util import *
from scipy.ndimage.filters import gaussian_filter1d

import matplotlib.pyplot as plt 
from matplotlib.lines import Line2D
import numpy as np
import os

class Coin:
    def __init__(self,network_status):
        self.p = 1.0 #current price 
        self.s = 1000.0 #supply
        strategy = "total" if network_status<0 else "dependent"
        self.bh = BondHolder(1,strategy)
        self.de = Delegate(1,strategy)
        self.no = NodeOperator(1,strategy,network_status)
 
        self.bond_store = [0]*MATURITY # to store BH's acquired bonds in each rebase
        self.mint_store = [0]*MATURITY # to store minted coins for each rebase      
        self.price_store = [1]*MATURITY # to store coin price for each rebase      
        self.price_data = {}
        self.roi_data = {}
        self.k1_data = {}
        self.k2_data = {}
        self.k3_data = {}
        self.interest_data = {}
        self.reward_data = {}

    def open_market(self,p_temp,i):
        self.p = p_temp + NOISE[i]

    def expand(self,rebase_id):
        minting = self.s*(self.p-PEG)/PEG #calculate minting amount for expansion
        self.s = self.s + minting #supply is expanded with minted coins
        self.mint_store[rebase_id%MATURITY] = minting # set minted coin for the rebase period
        p_temp = PEG #minting is distributed fully, hence p_temp = PEG
        return p_temp 

    def contract(self,rebase_id):
        open_bonds = self.s*(PEG-self.p)/PEG #calculate minting amount for expansion
        adopted = self.bh.acquire(open_bonds,self.de.k2)
        self.k1_data[rebase_id] = self.bh.k1
        self.s = self.s - adopted # supply contracted by BH's acquire
        self.bond_store[rebase_id%MATURITY] = adopted # set the bonds acquired by BH
        adopted = self.de.acquire(adopted,self.no.k3)
        self.k2_data[rebase_id] = self.de.k2
        adopted = self.no.acquire(adopted)
        self.k3_data[rebase_id] = self.no.k3

        return self.s*self.p/(self.s-adopted)

    def rebase(self,rebase_id):
        total_bonds = sum(self.bond_store)
        if(total_bonds>0):
            price_at_investing = self.price_store[rebase_id%MATURITY] # this is the price when investing 
            maturing_bonds = self.bond_store[rebase_id%MATURITY] #current maturing bonds
            bh_maturity_reward_value = sum([(coins*price*maturing_bonds*BH_REWARD_RATIO)/total_bonds for coins,price,bonds in zip(self.mint_store,self.price_store,self.bond_store)])
            bh_interest_gain_value = maturing_bonds*(self.p-price_at_investing)+maturing_bonds*INTEREST*self.p
            if bh_interest_gain_value>0:
                bh_maturity_gain = bh_maturity_reward_value/bh_interest_gain_value
                self.roi_data[rebase_id] = bh_maturity_gain

        self.price_store[rebase_id%MATURITY] = self.p # set price for this rebase
        self.price_data[rebase_id] = self.p

        self.s = self.s + self.bond_store[rebase_id%MATURITY] #matured bonds added to the supply

        if self.p > PEG:
            p_temp = self.expand(rebase_id)
        elif self.p < PEG:
            p_temp=self.contract(rebase_id)
        else:
            p_temp = PEG
            pass
        
        self.open_market(p_temp,rebase_id)

   
    def run(self):
        for i in range(PERIOD):
            self.rebase(i)

def price_plot(dir):
    colors = ["black","blue","orange","purple","olive"]
    bins=np.arange(0.5,1.2,0.01)
    for k,f in enumerate([-1,1,0.75,0.5,0.25]):
        data = np.array([])
        for sim in range(SIMS):
            filepath = f"./{dir}/kappa{f}/{sim}_price.npy"
            data = np.append(data,np.load(filepath))
        arr = np.array(data,dtype=float)
        # arr = remove_outliers_iqr(arr)
        hist, bins = np.histogram(arr,bins=bins)
        hist_norm = hist/SIMS
        if f >0:
            lab = f"$\\bar \\kappa$ = {f} (mean = {np.round(np.mean(arr),3)})"
        else:
            lab = f"total adoption (mean = {np.round(np.mean(arr),3)})"

        ysmoothed = gaussian_filter1d(hist_norm, sigma=2)
        plt.plot(bins[:-1]+0.005,ysmoothed,color=colors[k],label=lab)

        
    plt.xlabel("price")
    plt.ylabel("count")
    plt.axvline(0.95,alpha=0.5,linestyle=":")
    plt.axvline(1.05,alpha=0.5,linestyle=":")
    plt.axvline(1,alpha=0.5,linestyle="-")
    plt.legend()
    plt.savefig('price.png', dpi=300)
    plt.cla()


def roi_plot(dir):
    colors = ["black","blue","orange","purple","olive"]
    bins = np.arange(0,10,0.05)
    for k,f in enumerate([-1,1,0.75,0.5,0.25]):
        data = np.array([])
        for sim in range(SIMS):
            filepath = f"./{dir}/kappa{f}/{sim}_roi.npy"
            data = np.append(data,np.load(filepath))
        arr = np.array(data,dtype=float)
        print(arr.shape)
        arr = remove_outliers_iqr(arr)
        hist, bins = np.histogram(arr,bins=bins)
        hist_norm = hist/SIMS
        if f >0:
            lab = f"$\\bar \\kappa$ = {f} (mean = {np.round(np.mean(arr),3)})"
        else:
            lab = f"total adoption (mean = {np.round(np.mean(arr),3)})"

        ysmoothed = gaussian_filter1d(hist_norm, sigma=5)

        plt.plot(bins[:-1]+0.025,ysmoothed,color=colors[k],label=lab)

    plt.xlabel("reward:interest")
    plt.ylabel("count")
    plt.legend()
    plt.savefig('roi.png', dpi=300)
    plt.cla()



def k_plot(dir):
    lines = ['--',':','-']
    bins=np.arange(0,1,0.02)
    fig, axs = plt.subplots(nrows=1, ncols=4,figsize=(20,5),sharey=True)
    plt.subplots_adjust(hspace=0)

    fig.tight_layout()
    for i,f in enumerate([1,0.75,0.5,0.25]):
            k1_data = np.array([])
            k2_data = np.array([])
            k3_data = np.array([])
            for sim in range(SIMS):
                filepath = f"./{dir}/kappa{f}/{sim}_k1.npy"
                k1_data = np.append(k1_data,np.load(filepath))
                filepath = f"./{dir}/kappa{f}/{sim}_k2.npy"
                k2_data = np.append(k2_data,np.load(filepath))
                filepath = f"./{dir}/kappa{f}/{sim}_k3.npy"
                k3_data = np.append(k3_data,np.load(filepath))
        
            for k,data in enumerate([k1_data,k2_data,k3_data]):
                arr = np.array(data,dtype=float)
                hist, bins = np.histogram(arr,bins=bins)
                hist_norm = hist/SIMS
                axs[i].plot(bins[:-1]+0.01,hist_norm,linestyle=lines[k],color="gray",linewidth=4)
            axs[i].set_title(f"$\\bar \\kappa$ = {f}",fontsize=15)
    axs[0].tick_params(labelleft=True)  


    k1_line = Line2D([0,1],[0,1],linestyle=lines[0],color="gray",linewidth=4)
    k2_line = Line2D([0,1],[0,1],linestyle=lines[1],color="gray",linewidth=4)
    k3_line = Line2D([0,1],[0,1],linestyle=lines[2],color="gray",linewidth=4)

    plt.legend([ k1_line, k2_line, k3_line],['k1','k2','k3'],loc='upper right',fontsize=15)
    plt.subplots_adjust(bottom=0.1)
    plt.subplots_adjust(top=0.9)
    plt.subplots_adjust(left=0.04)

    fig.text(0.5, 0.01, 'adoption rates', ha='center',fontsize=15)
    fig.text(0, 0.5, 'count', va='center', rotation='vertical',fontsize=15)
    plt.savefig('kappa.png', dpi=300)
    plt.clf()

   
    # plt.axvline(1,alpha=0.5,linestyle="-")
    # plt.show()

def simulate(dir):
    #run simulations and store
    for k,f in enumerate([-1,1,0.75,0.5,0.25]):
        data = []
        os.makedirs(f"./{dir}/kappa{f}/", exist_ok=True)
        for sim in range(SIMS):
            c = Coin(f)
            c.run()
            roi_arr = np.array(list(c.roi_data.values()),dtype=float)
            roi_filepath = f"./{dir}/kappa{f}/{sim}_roi.npy"
            if not os.path.exists(roi_filepath):
                open(roi_filepath, 'w').close()
            np.save(roi_filepath,roi_arr)

            price_arr = np.array(list(c.price_data.values()),dtype=float)
            price_filepath = f"./{dir}/kappa{f}/{sim}_price.npy"
            if not os.path.exists(price_filepath):
                open(price_filepath, 'w').close()
            np.save(price_filepath,price_arr)


            k1_arr = np.array(list(c.k1_data.values()),dtype=float)
            k1_filepath = f"./{dir}/kappa{f}/{sim}_k1.npy"
            if not os.path.exists(k1_filepath):
                open(k1_filepath, 'w').close()
            np.save(k1_filepath,k1_arr)

            k2_arr = np.array(list(c.k2_data.values()),dtype=float)
            k2_filepath = f"./{dir}/kappa{f}/{sim}_k2.npy"
            if not os.path.exists(k2_filepath):
                open(k2_filepath, 'w').close()
            np.save(k2_filepath,k2_arr)

            k3_arr = np.array(list(c.k3_data.values()),dtype=float)
            k3_filepath = f"./{dir}/kappa{f}/{sim}_k3.npy"
            if not os.path.exists(k3_filepath):
                open(k3_filepath, 'w').close()
            np.save(k3_filepath,k3_arr)


def test():
    #run simulations and store
    for k,f in enumerate([1]):
        for sim in range(SIMS):
            c = Coin(f)
            c.run()


if __name__=="__main__": 
    # simulate("data90")
    # k_plot("data90")
    k_plot("data90")
    # price_plot("data90")
    # roi_plot("data90")
    # roi_plot("data90")
    # roi_plot("data365")
    # plot("data30")
    # simulate("data365")
   



   