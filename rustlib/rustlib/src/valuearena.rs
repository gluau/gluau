/// The Handle passed to Go
#[derive(Clone, Copy, PartialEq, Debug)]
#[repr(C)]
pub struct Handle {
    pub index: u32,      // Slot in the Vec
    pub generation: u32, // Unique version ID
}

/// A slot in the Arena, either occupied or free
enum Slot<T> {
    Occupied {
        generation: u32,
        value: T,
    },
    Free {
        next_free: u32, // Points to the next hole in the free list
    },
}

/// A generational arena for storing values with O(1) insert, get, and remove.
/// 
/// Uses generation counters to ensure safety against use-after-free.
pub struct Arena<T> {
    slots: Vec<Slot<T>>,
    first_free: Option<u32>, // Head of the free list
    generation_counters: Vec<u32>, // Persisted generations
}

impl<T> Arena<T> {
    pub fn new() -> Self {
        Self {
            slots: Vec::with_capacity(1024), // Pre-allocate to avoid early resizing
            first_free: None,
            generation_counters: Vec::with_capacity(1024),
        }
    }

    /// O(1) Allocation
    pub fn insert(&mut self, value: T) -> Handle {
        if let Some(idx) = self.first_free {
            // Reuse a hole (Free List) if possible
            let slot = &mut self.slots[idx as usize];
            
            // Extract the next pointer from the free slot
            let next_free = match slot {
                Slot::Free { next_free } => *next_free,
                _ => unreachable!("Corrupt free list"),
            };
            
            self.first_free = if next_free == u32::MAX { None } else { Some(next_free) };
            
            // Get the current generation for this slot
            let gene = self.generation_counters[idx as usize];
            
            *slot = Slot::Occupied { generation: gene, value };
            
            Handle { index: idx, generation: gene }
        } else {
            // 2. No holes? Append to the end.
            let idx = self.slots.len() as u32;
            let gene = 0; // First generation
            
            self.slots.push(Slot::Occupied { generation: gene, value });
            self.generation_counters.push(gene);
            
            Handle { index: idx, generation: gene }
        }
    }

    /// O(1) Lookup
    pub fn get(&self, handle: Handle) -> Option<&T> {
        // 1. Bounds Check
        let slot = self.slots.get(handle.index as usize)?;
        
        // 2. Generation Check (The "Magic" Safety)
        match slot {
            Slot::Occupied { generation, value } if *generation == handle.generation => {
                Some(value)
            }
            _ => None, // Either freed (Slot::Free) or reused (generation mismatch)
        }
    }

    /// O(1) Removal
    pub fn remove(&mut self, handle: Handle) -> Option<T> {
        if handle.index as usize >= self.slots.len() {
            return None;
        }

        // Check if valid
        let current_gen = self.generation_counters[handle.index as usize];
        if current_gen != handle.generation {
            return None;
        }

        // Increment generation so old handles become invalid
        self.generation_counters[handle.index as usize] += 1;

        // Extract value and swap to Free state
        let next_free = self.first_free.unwrap_or(u32::MAX);
        self.first_free = Some(handle.index);
        
        // Replace slot with Free marker
        let old_slot = std::mem::replace(
            &mut self.slots[handle.index as usize], 
            Slot::Free { next_free }
        );

        match old_slot {
            Slot::Occupied { value, .. } => Some(value),
            _ => unreachable!(),
        }
    }
}